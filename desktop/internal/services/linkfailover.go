package services

import (
	"sync"
	"time"
)

// A NIC that drops off the machine must not take the rules pointing at it down
// with it. Saved rules keep their chosen adapter; while that adapter is gone the
// rule is projected onto the aggregation pool, which by then only holds links
// that still work. Nothing here is persisted, so recovery — or restarting the
// app — restores the original routing with no user action.
//
// The projection is consulted from writeSingBoxRuleSetPlanLocked, which every
// routing publication goes through (TUN startup, rule edits, hot reload). That
// keeps the failover path identical to the normal one instead of adding a second
// writer for the watched rule-set files.
type outboundAvailability struct {
	mu          sync.RWMutex
	unavailable map[string]string
}

var liveOutbounds = &outboundAvailability{}

func (a *outboundAvailability) replace(next map[string]string) {
	copied := make(map[string]string, len(next))
	for id, reason := range next {
		if id == "" {
			continue
		}
		copied[id] = reason
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(copied) == 0 {
		a.unavailable = nil
		return
	}
	a.unavailable = copied
}

func (a *outboundAvailability) snapshot() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.unavailable) == 0 {
		return nil
	}
	copied := make(map[string]string, len(a.unavailable))
	for id, reason := range a.unavailable {
		copied[id] = reason
	}
	return copied
}

// projectRoutingRules re-points rules whose adapter is unreachable at the
// aggregation pool. Disabled rules are left alone so that enabling one later
// does not silently inherit a stale projection.
func projectRoutingRules(rules []RoutingRule) []RoutingRule {
	unavailable := liveOutbounds.snapshot()
	if len(unavailable) == 0 {
		return rules
	}
	projected := make([]RoutingRule, len(rules))
	copy(projected, rules)
	changed := false
	for i := range projected {
		if projected[i].Disabled {
			continue
		}
		if _, gone := unavailable[projected[i].Outbound]; !gone {
			continue
		}
		projected[i].Outbound = OutboundAggregation
		changed = true
	}
	if !changed {
		return rules
	}
	return projected
}

// projectOutbound redirects a single outbound onto the aggregation pool while
// its adapter is missing. The global default rule is validated and compiled on
// paths that never see projectRoutingRules, so it needs the same treatment —
// otherwise a user whose default egress is the dropped NIC cannot start
// aggregation at all, even though another line is working.
func projectOutbound(outbound string) string {
	if _, gone := liveOutbounds.snapshot()[outbound]; gone {
		return OutboundAggregation
	}
	return outbound
}

// linkCondition is one adapter's reachability as the failover loop sees it.
type linkCondition struct {
	ID      string
	Name    string
	Kind    string
	Present bool
	Reason  string
}

// LinkFailoverState is what the UI shows while a link is degraded. It is also
// the input for the home screen banner, so it stays free of internal handles.
type LinkFailoverState struct {
	Degraded bool   `json:"degraded"`
	Reason   string `json:"reason,omitempty"`
	// Note carries the one case the hot path cannot fix: route.final is pinned in
	// the running configuration, so a global default egress that points at the
	// dropped adapter only moves on the next start.
	Note           string    `json:"note,omitempty"`
	LostAdapters   []string  `json:"lost_adapters,omitempty"`
	ActiveAdapters []string  `json:"active_adapters,omitempty"`
	Recovering     bool      `json:"recovering,omitempty"`
	RecoverInMs    int       `json:"recover_in_ms,omitempty"`
	ChangedAt      time.Time `json:"changed_at,omitempty"`
}

// failoverGate serializes the degradation and restoration transactions so two
// simultaneous link events cannot interleave a projection with the opposite
// scheduling update.
var failoverGate sync.Mutex
