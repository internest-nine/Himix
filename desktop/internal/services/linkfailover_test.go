package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func linkTestService(t *testing.T, tracked []AdapterView, live []AdapterView) (*EngineService, *[]map[string]any) {
	t.Helper()
	t.Setenv("HIMIX_DATA_DIR", t.TempDir())
	settings := NewSettingsService()
	next := settings.Get()
	next.Mode = "tun"
	next.Strategy = "round-robin"
	next.SelectedAdapterIDs = make([]string, 0, len(tracked))
	for _, adapter := range tracked {
		next.SelectedAdapterIDs = append(next.SelectedAdapterIDs, adapter.ID)
	}
	if _, err := settings.Update(next); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	calls := &[]map[string]any{}
	service := &EngineService{
		settings:              settings,
		runtimeBindings:       cloneRuntimeBindings(tracked),
		runtimeBindingEnabled: true,
		runtimeBindingMode:    "tun",
		runtimeAdapters: func() ([]AdapterView, error) {
			return live, nil
		},
		linkRequest: func(_ context.Context, method string, params any, result any) error {
			if method != "engine.scheduling" {
				t.Fatalf("unexpected method %q", method)
			}
			*calls = append(*calls, params.(map[string]any))
			if raw, ok := result.(*json.RawMessage); ok {
				*raw = json.RawMessage(`{}`)
			}
			return nil
		},
	}
	t.Cleanup(func() { liveOutbounds.replace(nil) })
	return service, calls
}

func wifiAndWired() (AdapterView, AdapterView) {
	return AdapterView{ID: "WiFi", Name: "Wi-Fi", Kind: "wifi", Address: "192.168.0.9", IfIndex: 5, Operational: true},
		AdapterView{ID: "Ethernet", Name: "以太网", Kind: "ethernet", Address: "10.176.8.14", IfIndex: 7, Operational: true}
}

func TestProjectRoutingRulesIsANoOpWhileEveryLinkIsUp(t *testing.T) {
	rules := []RoutingRule{
		{MatchType: MatchDomain, Value: "example.com", Outbound: "nic_WiFi"},
		{MatchType: MatchIP, Value: "10.0.0.1", Outbound: OutboundAggregation},
	}
	liveOutbounds.replace(nil)
	projected := projectRoutingRules(rules)
	if len(projected) != len(rules) {
		t.Fatalf("unexpected length %d", len(projected))
	}
	for i := range rules {
		if projected[i] != rules[i] {
			t.Fatalf("rule %d was rewritten while every link was up", i)
		}
	}
}

func TestProjectRoutingRulesMovesOnlyRulesOnMissingAdapters(t *testing.T) {
	rules := []RoutingRule{
		{MatchType: MatchDomain, Value: "intranet.example", Outbound: "nic_WiFi"},
		{MatchType: MatchDomain, Value: "public.example", Outbound: "nic_Ethernet"},
		{MatchType: MatchDomain, Value: "disabled.example", Outbound: "nic_WiFi", Disabled: true},
		{MatchType: MatchDomain, Value: "direct.example", Outbound: "direct"},
	}
	// A disabled rule keeps its adapter: turning it on later must not inherit a
	// projection that no longer applies.
	liveOutbounds.replace(map[string]string{"nic_WiFi": "Wi-Fi 已断开"})
	t.Cleanup(func() { liveOutbounds.replace(nil) })

	projected := projectRoutingRules(rules)
	if projected[0].Outbound != OutboundAggregation {
		t.Fatalf("missing adapter was not projected: %q", projected[0].Outbound)
	}
	if projected[1].Outbound != "nic_Ethernet" {
		t.Fatalf("healthy adapter was rewritten: %q", projected[1].Outbound)
	}
	if projected[2].Outbound != "nic_WiFi" {
		t.Fatalf("disabled rule was rewritten: %q", projected[2].Outbound)
	}
	if projected[3].Outbound != "direct" {
		t.Fatalf("non-adapter outbound was rewritten: %q", projected[3].Outbound)
	}
	if rules[0].Outbound != "nic_WiFi" {
		t.Fatal("the source rules were mutated")
	}
}

func TestDegradeLiveLinksShrinksThePoolOnceAndRestoresAfterTheHold(t *testing.T) {
	wifi, wired := wifiAndWired()
	service, calls := linkTestService(t, []AdapterView{wifi, wired}, []AdapterView{wired})

	service.degradeLiveLinks([]AdapterView{wifi}, []AdapterView{wired})
	if len(*calls) != 1 {
		t.Fatalf("expected one scheduling update, got %d", len(*calls))
	}
	adapters := (*calls)[0]["adapters"].([]map[string]any)
	if len(adapters) != 1 || adapters[0]["name"] != "以太网" {
		t.Fatalf("pool was not shrunk to the surviving link: %#v", adapters)
	}
	state := service.LinkFailover()
	if !state.Degraded || len(state.LostAdapters) != 1 || state.LostAdapters[0] != "Wi-Fi" {
		t.Fatalf("unexpected state: %#v", state)
	}

	// The scan repeats every tick. Repeating the same loss must not multiply the
	// requests, otherwise a long outage would spam the core.
	service.degradeLiveLinks([]AdapterView{wifi}, []AdapterView{wired})
	if len(*calls) != 1 {
		t.Fatalf("repeated scan re-applied the update: %d", len(*calls))
	}

	// Recovery starts a countdown but must not restore anything before it ends.
	service.restoreLiveLinks([]AdapterView{wifi, wired})
	if len(*calls) != 1 {
		t.Fatal("restored before the recovery hold elapsed")
	}
	if state := service.LinkFailover(); !state.Recovering || state.RecoverInMs <= 0 {
		t.Fatalf("expected a pending recovery: %#v", state)
	}

	// Fast-forward past the hold instead of sleeping through it.
	service.mu.Lock()
	service.linkRecoverSince = time.Now().Add(-liveLinkRecoveryHold - time.Second)
	service.mu.Unlock()

	service.restoreLiveLinks([]AdapterView{wifi, wired})
	if len(*calls) != 2 {
		t.Fatalf("expected a restore update, got %d calls", len(*calls))
	}
	restored := (*calls)[1]["adapters"].([]map[string]any)
	if len(restored) != 2 {
		t.Fatalf("pool was not restored: %#v", restored)
	}
	if state := service.LinkFailover(); state.Degraded {
		t.Fatalf("still degraded after recovery: %#v", state)
	}
	if len(liveOutbounds.snapshot()) != 0 {
		t.Fatal("projection survived the recovery")
	}
}

func TestDegradeLiveLinksKeepsTheProjectionWhenEveryLinkIsGone(t *testing.T) {
	wifi, wired := wifiAndWired()
	service, calls := linkTestService(t, []AdapterView{wifi, wired}, nil)

	service.degradeLiveLinks([]AdapterView{wifi, wired}, nil)
	// An empty pool would be rejected by the core and leave the last update in
	// place, so no scheduling request may be sent.
	if len(*calls) != 0 {
		t.Fatalf("sent a scheduling update with no surviving link: %d", len(*calls))
	}
	projected := liveOutbounds.snapshot()
	for _, outbound := range []string{"nic_WiFi", "nic_Ethernet", "nic_wifi", "nic_ethernet"} {
		if _, ok := projected[outbound]; !ok {
			t.Fatalf("%s was not projected: %#v", outbound, projected)
		}
	}
	state := service.LinkFailover()
	if !state.Degraded || len(state.ActiveAdapters) != 0 {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestDegradeLiveLinksCancelsAPendingRecovery(t *testing.T) {
	wifi, wired := wifiAndWired()
	service, _ := linkTestService(t, []AdapterView{wifi, wired}, []AdapterView{wired})

	service.degradeLiveLinks([]AdapterView{wifi}, []AdapterView{wired})
	service.restoreLiveLinks([]AdapterView{wifi, wired})
	if !service.LinkFailover().Recovering {
		t.Fatal("recovery should be pending")
	}
	// The adapter flaps back out: the countdown has to be discarded, otherwise
	// the banner would claim a recovery that can no longer happen.
	service.degradeLiveLinks([]AdapterView{wifi}, []AdapterView{wired})
	if state := service.LinkFailover(); state.Recovering {
		t.Fatalf("stale recovery countdown survived a new loss: %#v", state)
	}
}

func TestClearLiveLinkFailoverDropsEveryOverride(t *testing.T) {
	wifi, wired := wifiAndWired()
	service, _ := linkTestService(t, []AdapterView{wifi, wired}, []AdapterView{wired})
	service.degradeLiveLinks([]AdapterView{wifi}, []AdapterView{wired})

	service.clearLiveLinkFailover()
	if len(liveOutbounds.snapshot()) != 0 {
		t.Fatal("projection survived a session teardown")
	}
	if state := service.LinkFailover(); state.Degraded {
		t.Fatalf("session ended while still degraded: %#v", state)
	}
}

// reconcileLiveLinks is the path the real loop takes; the cases above drive the
// transactions directly. This covers adapter enumeration, the "seen before"
// guard and the unticked-adapter filter.
func TestReconcileLiveLinksDegradesOnlyForAdaptersThisSessionSaw(t *testing.T) {
	wifi, wired := wifiAndWired()
	t.Setenv("HIMIX_DATA_DIR", t.TempDir())
	settings := NewSettingsService()
	next := settings.Get()
	next.Mode = "tun"
	next.Strategy = "round-robin"
	next.SelectedAdapterIDs = []string{wifi.ID, wired.ID}
	if _, err := settings.Update(next); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	live := []AdapterView{wifi, wired}
	adapterService := NewAdapterService(settings)
	adapterService.list = func() ([]AdapterView, error) { return live, nil }
	calls := []map[string]any{}
	service := &EngineService{
		settings:              settings,
		adapters:              adapterService,
		runtimeBindings:       []AdapterView{wifi, wired},
		runtimeBindingEnabled: true,
		runtimeBindingMode:    "tun",
		linkRequest: func(_ context.Context, _ string, params any, _ any) error {
			calls = append(calls, params.(map[string]any))
			return nil
		},
	}
	t.Cleanup(func() { liveOutbounds.replace(nil) })

	// First scan only establishes the baseline.
	service.reconcileLiveLinks()
	if len(calls) != 0 || service.LinkFailover().Degraded {
		t.Fatalf("first scan acted on a healthy set: %d calls", len(calls))
	}

	live = []AdapterView{wired}
	service.reconcileLiveLinks()
	if len(calls) != 1 {
		t.Fatalf("expected one scheduling update, got %d", len(calls))
	}
	if !service.LinkFailover().Degraded {
		t.Fatal("loss was not reported")
	}

	// The same scan repeats every tick; it must stay quiet.
	service.reconcileLiveLinks()
	if len(calls) != 1 {
		t.Fatalf("repeated scan re-applied the update: %d", len(calls))
	}
}

func TestReconcileLiveLinksSkipsAdaptersTheUserUnticked(t *testing.T) {
	wifi, wired := wifiAndWired()
	t.Setenv("HIMIX_DATA_DIR", t.TempDir())
	settings := NewSettingsService()
	next := settings.Get()
	next.Mode = "tun"
	next.Strategy = "round-robin"
	// The runtime bindings still list the wired adapter, but the user has since
	// unticked it: sending it back to the core would route metered traffic over
	// an adapter the UI shows as off.
	next.SelectedAdapterIDs = []string{wifi.ID}
	if _, err := settings.Update(next); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	adapterService := NewAdapterService(settings)
	adapterService.list = func() ([]AdapterView, error) {
		return []AdapterView{wifi, wired}, nil
	}
	calls := []map[string]any{}
	service := &EngineService{
		settings:              settings,
		adapters:              adapterService,
		runtimeBindings:       []AdapterView{wifi, wired},
		runtimeBindingEnabled: true,
		runtimeBindingMode:    "tun",
		linkRequest: func(_ context.Context, _ string, params any, _ any) error {
			calls = append(calls, params.(map[string]any))
			return nil
		},
	}
	t.Cleanup(func() { liveOutbounds.replace(nil) })

	service.reconcileLiveLinks()
	if len(calls) != 0 {
		t.Fatalf("unticked adapter triggered a switch: %d calls", len(calls))
	}
	if service.LinkFailover().Degraded {
		t.Fatal("unticked adapter was treated as an outage")
	}
}

func TestReconcileLiveLinksIgnoresBindingsTheHostNeverEnumerated(t *testing.T) {
	t.Setenv("HIMIX_DATA_DIR", t.TempDir())
	settings := NewSettingsService()
	next := settings.Get()
	next.Mode = "tun"
	next.SelectedAdapterIDs = []string{"loopback"}
	if _, err := settings.Update(next); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	wifi, _ := wifiAndWired()
	adapterService := NewAdapterService(settings)
	adapterService.list = func() ([]AdapterView, error) { return []AdapterView{wifi}, nil }
	calls := 0
	service := &EngineService{
		settings:              settings,
		adapters:              adapterService,
		runtimeBindings:       []AdapterView{{ID: "loopback", Name: "loopback", Kind: "ethernet", Operational: true}},
		runtimeBindingEnabled: true,
		linkRequest: func(_ context.Context, _ string, _ any, _ any) error {
			calls++
			return nil
		},
	}
	t.Cleanup(func() { liveOutbounds.replace(nil) })

	// A binding that was never observed live is a fixture artifact, not a cable
	// being pulled.
	service.reconcileLiveLinks()
	if calls != 0 || service.LinkFailover().Degraded {
		t.Fatalf("an unobserved binding was reported as an outage: %d calls", calls)
	}
}

func TestApplyLiveLinkSchedulingReportsFailureSoTheSwitchIsNotAnnounced(t *testing.T) {
	wifi, wired := wifiAndWired()
	service, _ := linkTestService(t, []AdapterView{wifi, wired}, []AdapterView{wired})
	service.linkRequest = func(_ context.Context, _ string, _ any, _ any) error {
		return errors.New("core busy")
	}

	service.degradeLiveLinks([]AdapterView{wifi}, []AdapterView{wired})
	if service.LinkFailover().Degraded {
		t.Fatal("announced a switch that the core rejected")
	}
	if len(liveOutbounds.snapshot()) != 0 {
		t.Fatal("projected rules without a matching pool")
	}

	// The next tick has to try again rather than stay stuck on the same loss.
	service.linkRequest = func(_ context.Context, _ string, params any, _ any) error {
		if params == nil {
			t.Fatal("empty request")
		}
		return nil
	}
	service.degradeLiveLinks([]AdapterView{wifi}, []AdapterView{wired})
	if !service.LinkFailover().Degraded {
		t.Fatal("retry did not take effect")
	}
}
