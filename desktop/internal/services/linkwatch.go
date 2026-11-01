package services

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

// Link failover keeps aggregation usable when a NIC disappears. The saved
// configuration never changes: the aggregation pool is shrunk to the links that
// still work, and rules pointing at the missing adapter are projected onto that
// pool, so traffic keeps flowing instead of timing out on a dead source address.
//
// Both actions are hot: engine.scheduling mutates the pool without touching the
// listeners, and the projected rules are published through the watched rule-set
// files, which sing-box reloads in place. The engine is never restarted here,
// which is what keeps established connections on the surviving link alive.
//
// Degradation is immediate because it only removes a path that cannot work.
// Recovery waits liveLinkRecoveryHold so a flapping adapter cannot drag traffic
// back and forth every second.
const (
	liveLinkPollInterval  = 1500 * time.Millisecond
	liveLinkRecoveryHold  = 5 * time.Second
	liveLinkSettleDelay   = 250 * time.Millisecond
	liveLinkDebounce      = 200 * time.Millisecond
	liveLinkRequestBudget = 3 * time.Second
)

// watchLiveLinks owns the failover loop for one engine session. It exits when
// the session ends, so a stopped engine neither scans nor writes rule-sets.
//
// Interface notifications drive the fast path; the ticker is the safety net for
// a link that stops forwarding without telling Windows it is gone, such as
// walking out of Wi-Fi range while the adapter keeps its lease.
func (s *EngineService) watchLiveLinks(done <-chan struct{}) {
	changes, stopNotifications := watchLinkNotifications()
	defer stopNotifications()
	ticker := time.NewTicker(liveLinkPollInterval)
	defer ticker.Stop()
	// A short settle keeps startup traffic from racing the first observation.
	select {
	case <-done:
		return
	case <-time.After(liveLinkSettleDelay):
	}
	s.reconcileLiveLinks()
	// One Windows action raises several notifications; collapse them so the scan
	// runs once the system has finished applying the change.
	var debounce <-chan time.Time
	for {
		select {
		case <-done:
			s.clearLiveLinkFailover()
			return
		case <-ticker.C:
			s.reconcileLiveLinks()
		case <-changes:
			if debounce == nil {
				debounce = time.After(liveLinkDebounce)
			}
		case <-debounce:
			debounce = nil
			s.reconcileLiveLinks()
		}
	}
}

// liveAdapterSet is the set of adapters that currently carry a usable address.
// A disabled Wi-Fi adapter leaves net.Interfaces immediately, which is what
// makes switching away from a manually disabled link fast.
//
// This reads the OS through AdapterService rather than the engine's runtime
// snapshot on purpose: the question here is what Windows still sees.
func (s *EngineService) liveAdapterSet() (map[string]AdapterView, error) {
	if s.adapters == nil {
		return nil, errors.New("网卡服务不可用")
	}
	available, err := s.adapters.List()
	if err != nil {
		return nil, err
	}
	live := make(map[string]AdapterView, len(available))
	for _, adapter := range available {
		if adapter.IsVirtual || !usableRuntimeBinding(adapter) {
			continue
		}
		live[adapter.ID] = adapter
	}
	return live, nil
}

// trackedBindings are the adapters this session was started against, including
// ones that have since disappeared. Kind is only known from this snapshot once
// an adapter is gone, so the projections below rely on it.
//
// runtimeBindings is append-only and also records the DNS egress adapter, so it
// is filtered against the persisted selection: an adapter the user unticked must
// never be handed back to the core.
func (s *EngineService) trackedBindings() []AdapterView {
	s.mu.Lock()
	enabled := s.runtimeBindingEnabled
	tracked := cloneRuntimeBindings(s.runtimeBindings)
	s.mu.Unlock()
	if !enabled {
		return nil
	}
	selected := s.settings.Get().SelectedAdapterIDs
	if len(selected) == 0 {
		return tracked
	}
	wanted := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		wanted[id] = struct{}{}
	}
	filtered := tracked[:0]
	for _, adapter := range tracked {
		if _, ok := wanted[adapter.ID]; ok {
			filtered = append(filtered, adapter)
		}
	}
	return filtered
}

func (s *EngineService) reconcileLiveLinks() {
	tracked := s.trackedBindings()
	if len(tracked) == 0 {
		return
	}
	live, err := s.liveAdapterSet()
	if err != nil {
		// A scan that cannot enumerate adapters is a host-level problem, not
		// evidence that every link died.
		return
	}
	// Only an adapter this session has actually seen can be reported missing.
	// Bindings can name interfaces the OS layer here never enumerates (loopback
	// fixtures in tests), and those are not outages.
	lost := make([]AdapterView, 0, len(tracked))
	active := make([]AdapterView, 0, len(tracked))
	for _, adapter := range tracked {
		if current, ok := live[adapter.ID]; ok {
			s.markLinkKnown(adapter.ID, len(live))
			active = append(active, current)
			continue
		}
		if !s.linkWasSeen(adapter.ID) {
			continue
		}
		lost = append(lost, adapter)
	}
	if len(lost) > 0 {
		s.degradeLiveLinks(lost, active)
		return
	}
	s.restoreLiveLinks(active)
}

// degradeLiveLinks shrinks the pool and projects the rules. It is idempotent:
// the repeated scans that happen while a user is disabling an adapter must not
// multiply the scheduling requests or rewrite the rule-set files again.
func (s *EngineService) degradeLiveLinks(lost, active []AdapterView) {
	signature := linkPoolSignature(lost, active)
	s.mu.Lock()
	same := s.linkDegraded && s.linkLossSignature == signature
	// A new loss also cancels any recovery countdown a brief reconnection had
	// started, otherwise the banner would claim a switch-back that cannot happen.
	s.linkRecoverSince = time.Time{}
	s.mu.Unlock()
	if same {
		return
	}

	failoverGate.Lock()
	defer failoverGate.Unlock()

	// Shrink first. Rules that start pointing at aggregation while the pool still
	// holds the dead link would be steered straight back onto it.
	if len(active) > 0 && !s.applyLiveLinkScheduling(active) {
		// Without a clean pool the projection would be a lie. Leave everything as
		// it is and let the next tick retry instead of announcing a switch.
		s.recordLiveLinkEvent("link_degrade_retry", map[string]any{"lost": adapterNames(lost)})
		return
	}

	liveOutbounds.replace(unavailableOutbounds(lost, active))
	s.publishProjectedRules()

	settings := s.settings.Get()
	s.mu.Lock()
	s.linkDegraded = true
	s.linkLossSignature = signature
	s.linkDefaultEgressStale = projectOutbound(settings.DefaultOutbound) != settings.DefaultOutbound
	s.linkLostAdapters = adapterNames(lost)
	s.linkActiveAdapters = adapterNames(active)
	s.linkSwitchAt = time.Now()
	s.mu.Unlock()

	s.recordLiveLinkEvent("link_degraded", map[string]any{
		"lost": adapterNames(lost), "active": adapterNames(active),
	})
}

// restoreLiveLinks waits for the survivor set to hold steady before handing the
// full pool back. The wait is what stops an adapter that reconnects for a second
// from pulling every new flow back onto it.
func (s *EngineService) restoreLiveLinks(active []AdapterView) {
	s.mu.Lock()
	degraded := s.linkDegraded
	since := s.linkRecoverSince
	if !degraded {
		s.mu.Unlock()
		return
	}
	if since.IsZero() {
		s.linkRecoverSince = time.Now()
		s.mu.Unlock()
		return
	}
	remaining := liveLinkRecoveryHold - time.Since(since)
	s.mu.Unlock()
	if remaining > 0 {
		return
	}
	if len(active) == 0 {
		// Still nothing to route over; keep the degraded projection in place.
		return
	}

	failoverGate.Lock()
	defer failoverGate.Unlock()
	// Restore the pool first: the rules are about to point back at per-adapter
	// channels, which do not depend on the pool.
	if !s.applyLiveLinkScheduling(active) {
		s.recordLiveLinkEvent("link_restore_retry", map[string]any{"active": adapterNames(active)})
		return
	}
	liveOutbounds.replace(nil)
	s.publishProjectedRules()

	s.mu.Lock()
	s.linkDegraded = false
	s.linkLossSignature = ""
	s.linkDefaultEgressStale = false
	s.linkLostAdapters = nil
	s.linkActiveAdapters = adapterNames(active)
	s.linkSwitchAt = time.Now()
	s.linkRecoverSince = time.Time{}
	s.mu.Unlock()

	s.recordLiveLinkEvent("link_restored", map[string]any{"active": adapterNames(active)})
}

// unavailableOutbounds maps every saved outbound that cannot currently be used
// onto the reason shown in the log. The per-adapter keys are what rules
// reference; the family keys cover the global default egress.
func unavailableOutbounds(lost, active []AdapterView) map[string]string {
	unavailable := make(map[string]string, len(lost)+2)
	for _, adapter := range lost {
		unavailable["nic_"+adapter.ID] = adapter.Name + " 已断开"
	}
	activeIDs := make(map[string]struct{}, len(active))
	for _, adapter := range active {
		activeIDs[adapter.ID] = struct{}{}
	}
	for _, family := range []struct{ outbound, kind, label string }{
		{"nic_wifi", "wifi", "无线网络"},
		{"nic_ethernet", "ethernet", "有线网络"},
	} {
		gone, present := false, false
		for _, adapter := range lost {
			if adapter.Kind == family.kind {
				gone = true
			}
		}
		for _, adapter := range active {
			if adapter.Kind == family.kind {
				present = true
			}
		}
		if gone && !present {
			unavailable[family.outbound] = family.label + " 全部断开"
		}
	}
	return unavailable
}

// linkPoolSignature identifies the exact state a successful switch produced.
// Addresses and interface indexes are part of it because a surviving adapter
// that renews its lease needs to be re-bound in the core, even though the set of
// missing adapters did not change.
func linkPoolSignature(lost, active []AdapterView) string {
	var builder strings.Builder
	for _, adapter := range lost {
		builder.WriteString("L:")
		builder.WriteString(adapter.ID)
		builder.WriteString("|")
	}
	for _, adapter := range active {
		builder.WriteString("A:")
		builder.WriteString(adapter.ID)
		builder.WriteString("@")
		builder.WriteString(adapter.Address)
		builder.WriteString("#")
		builder.WriteString(adapter.SourceIPv6)
		builder.WriteString("|")
	}
	return builder.String()
}

// publishProjectedRules republishes the watched rule-set files. When TUN has not
// been started there is no manifest and the next start picks the projection up
// from writeSingBoxRuleSetPlanLocked, so an error here is never fatal.
//
// routingMutationMu keeps a concurrent rule edit from being overwritten with the
// snapshot read before it landed: the editor holds the same lock when it writes.
func (s *EngineService) publishProjectedRules() {
	routingMutationMu.Lock()
	defer routingMutationMu.Unlock()
	settings := s.settings.Get()
	if err := refreshSingBoxRuleSets(settings.RoutingRules, settings.RuleSets); err != nil && s.logs != nil {
		s.logs.RecordEvent("link_failover", "rule_projection_failed", map[string]any{"error": err.Error()})
	}
}

// applyLiveLinkScheduling swaps the aggregation pool for the surviving links.
// Established flows keep their binding, so this only steers new connections.
// It reports success so the caller never announces a switch that did not happen.
func (s *EngineService) applyLiveLinkScheduling(active []AdapterView) bool {
	if len(active) == 0 {
		return false
	}
	settings := s.settings.Get()
	strategy, err := normalizeSchedulingStrategy(settings.Strategy, settings.Weighted)
	if err != nil {
		return false
	}
	send := s.linkRequest
	if send == nil {
		if s.client == nil {
			return false
		}
		// A core that is not running has no pool to shrink; the next start reads
		// the projection instead.
		if !slices.Contains(s.client.Hello().Capabilities, "engine.scheduling") {
			return false
		}
		send = s.client.Request
	}
	if err := validateAdapterSources(active); err != nil {
		s.recordLiveLinkEvent("link_scheduling_rejected", map[string]any{"error": err.Error()})
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), liveLinkRequestBudget)
	defer cancel()
	var previous json.RawMessage
	request := map[string]any{
		"strategy": strategy,
		"weighted": strategy == "weighted",
		"adapters": engineAdapters(active),
	}
	if err := send(ctx, "engine.scheduling", request, &previous); err != nil {
		s.recordLiveLinkEvent("link_scheduling_failed", map[string]any{
			"error": err.Error(), "adapters": adapterNames(active),
		})
		return false
	}
	// Keep the restart path consistent with what is running right now.
	s.rememberAdditionalRuntimeBindings(active)
	return true
}

// clearLiveLinkFailover drops every runtime override. Called when the session
// ends so the next start reads the saved rules and pool verbatim.
func (s *EngineService) clearLiveLinkFailover() {
	failoverGate.Lock()
	defer failoverGate.Unlock()
	hadProjection := len(liveOutbounds.snapshot()) > 0
	liveOutbounds.replace(nil)
	s.resetLiveLinkState()
	if hadProjection {
		s.publishProjectedRules()
	}
}

// resetLiveLinkState forgets the runtime view without touching the published
// files. Stop uses it so the next start does not inherit the previous outage,
// which would otherwise show a stale banner and keep rules projected.
func (s *EngineService) resetLiveLinkState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.linkDegraded = false
	s.linkLossSignature = ""
	s.linkLostAdapters = nil
	s.linkActiveAdapters = nil
	s.linkSwitchAt = time.Time{}
	s.linkRecoverSince = time.Time{}
}

// LinkFailover reports the current degradation for the UI banner.
func (s *EngineService) LinkFailover() LinkFailoverState {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := LinkFailoverState{
		Degraded:       s.linkDegraded,
		LostAdapters:   append([]string(nil), s.linkLostAdapters...),
		ActiveAdapters: append([]string(nil), s.linkActiveAdapters...),
		ChangedAt:      s.linkSwitchAt,
	}
	if s.linkDegraded {
		if len(state.LostAdapters) > 0 {
			state.Reason = strings.Join(state.LostAdapters, "、") + " 已断开，流量已转到可用线路"
		} else {
			state.Reason = "部分线路已断开，流量已转到可用线路"
		}
		if s.linkDefaultEgressStale {
			state.Note = "全局默认出口设在了刚断开的网卡上，它要等下次启动加速才会改走其他线路；规则里的分流已经切过来了"
		}
		if !s.linkRecoverSince.IsZero() {
			state.Recovering = true
			remaining := liveLinkRecoveryHold - time.Since(s.linkRecoverSince)
			if remaining < 0 {
				remaining = 0
			}
			state.RecoverInMs = int(remaining / time.Millisecond)
		}
	}
	return state
}

// markLinkKnown records that this session has seen the adapter. The map can be
// cleared by a session teardown, so it is re-created on demand.
func (s *EngineService) markLinkKnown(id string, capacity int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.linkKnownAdapters == nil {
		s.linkKnownAdapters = make(map[string]bool, capacity)
	}
	s.linkKnownAdapters[id] = true
}

func (s *EngineService) linkWasSeen(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.linkKnownAdapters[id]
}

func (s *EngineService) recordLiveLinkEvent(event string, fields map[string]any) {
	if s.logs != nil {
		s.logs.RecordEvent("link_failover", event, fields)
	}
}

func adapterNames(adapters []AdapterView) []string {
	names := make([]string, 0, len(adapters))
	for _, adapter := range adapters {
		names = append(names, adapter.Name)
	}
	return names
}
