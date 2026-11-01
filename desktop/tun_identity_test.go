package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The engine creates the TUN adapter under one name, and the desktop decides
// which adapters to hide (and which routes belong to us) by matching that same
// name. When the two disagree the adapter becomes user-selectable as an
// ordinary uplink — the user can put our own tunnel into the aggregation pool —
// and ownTUN() stops recognising the routes we created. Both sides therefore
// have to agree on the literal, and this test pins it.
func TestEngineAndDesktopAgreeOnTunAdapterName(t *testing.T) {
	engineSource, err := os.ReadFile("../engine/internal/tun/supervisor.go")
	if err != nil {
		t.Fatal(err)
	}
	engineMatch := regexp.MustCompile(`tunInterfaceName\s*=\s*"([^"]+)"`).FindStringSubmatch(string(engineSource))
	if engineMatch == nil {
		t.Fatal("engine does not declare tunInterfaceName")
	}
	engineName := engineMatch[1]

	adaptersSource, err := os.ReadFile("internal/services/adapters.go")
	if err != nil {
		t.Fatal(err)
	}
	adapterMatch := regexp.MustCompile(`func isHimixManagedAdapter\(name string\) bool \{\s*return strings\.EqualFold\(strings\.TrimSpace\(name\), "([^"]+)"\)`).FindStringSubmatch(string(adaptersSource))
	if adapterMatch == nil {
		t.Fatal("desktop does not declare the managed adapter name")
	}
	desktopName := adapterMatch[1]

	if engineName != desktopName {
		t.Fatalf("engine creates the TUN as %q but the desktop hides %q; the adapter would leak into the uplink list", engineName, desktopName)
	}

	// ownTUN() must use the same literal, otherwise route ownership and
	// cleanup target a name nobody creates.
	routesSource, err := os.ReadFile("internal/services/network_routes.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(routesSource), `strings.EqualFold(r.Alias, "`+engineName+`")`) {
		t.Fatalf("ownTUN() does not match the engine TUN name %q", engineName)
	}

	// The stale-adapter classifier keys off the message the cleanup script
	// throws, so the two files are coupled through their text.
	cleanupSource, err := os.ReadFile("../engine/internal/tun/cleanup_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cleanupSource), "stale "+engineName+" device still exists") {
		t.Fatalf("cleanup script does not name the %q adapter in its stale-device error", engineName)
	}
	serverSource, err := os.ReadFile("../engine/internal/server/server.go")
	if err != nil {
		t.Fatal(err)
	}
	lowered := strings.ToLower("stale " + engineName + " device still exists")
	if !strings.Contains(string(serverSource), `"`+lowered+`"`) {
		t.Fatalf("stale-adapter classifier does not match the cleanup error %q", lowered)
	}
}
