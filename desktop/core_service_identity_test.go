package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The installer and the engine must agree, byte for byte, on where the Core
// Service lives and what it is called. The engine refuses to register the
// service unless its own executable sits at exactly the path the installer
// stages, and the desktop refuses to talk to a service whose name or pipe
// differs. A rename that updates only one side therefore produces an
// installer that installs successfully and then fails at service
// registration, which is exactly what happened when the product was renamed
// from HypoMux to Himix.
func TestInstallerAndEngineAgreeOnCoreServiceIdentity(t *testing.T) {
	scriptData, err := os.ReadFile("build/windows/nsis/project.nsi")
	if err != nil {
		t.Fatal(err)
	}
	script := string(scriptData)

	engineData, err := os.ReadFile("../engine/cmd/himix-engine/product_identity_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	engine := string(engineData)

	enginePolicyData, err := os.ReadFile("../engine/cmd/himix-engine/service_policy_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	enginePolicy := string(enginePolicyData)

	engineServiceData, err := os.ReadFile("../engine/cmd/himix-engine/service_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	engineService := string(engineServiceData)

	desktopServiceData, err := os.ReadFile("internal/engineclient/service_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	desktopService := string(desktopServiceData)

	// The engine keeps every product-name literal in one file so the two
	// sides cannot drift apart unnoticed.
	for _, constant := range []string{
		`productDirectoryName = "Himix"`,
		`engineExecutableName = "himix-engine.exe"`,
		`desktopExecutableName = "himix.exe"`,
		`coreServiceDisplayName = "Himix Core Service"`,
	} {
		if !strings.Contains(engine, constant) {
			t.Fatalf("engine identity constants are missing %q", constant)
		}
	}

	// The installer stages the protected payload under the same directory
	// name the engine insists on.
	protectedRoot := installerDefine(t, script, "HIMIX_PROTECTED_CORE_ROOT")
	if !strings.Contains(protectedRoot, `Himix`) {
		t.Fatalf("installer protected root does not use the Himix directory: %q", protectedRoot)
	}
	if !strings.Contains(enginePolicy, `filepath.Join(programData, productDirectoryName, "Core", "bin", engineExecutableName)`) {
		t.Fatal("engine does not derive its expected service path from the shared identity constants")
	}

	// The service name must be identical across the installer, the engine and
	// the desktop client, otherwise the desktop cannot connect.
	serviceName := installerDefine(t, script, "HIMIX_CORE_SERVICE")
	if !strings.Contains(engineService, `coreServiceName            = "`+serviceName+`"`) {
		t.Fatalf("engine service name does not match the installer (%q)", serviceName)
	}
	if !strings.Contains(desktopService, `CoreServiceName     = "`+serviceName+`"`) {
		t.Fatalf("desktop service name does not match the installer (%q)", serviceName)
	}

	// The named pipe is the desktop/engine handshake channel.
	enginePipe := firstPipeName(t, engineService)
	desktopPipe := firstPipeName(t, desktopService)
	if enginePipe == "" || enginePipe != desktopPipe {
		t.Fatalf("pipe name mismatch: engine=%q desktop=%q", enginePipe, desktopPipe)
	}
	if strings.Contains(enginePipe, "hypomux") {
		t.Fatalf("pipe name still uses the retired product name: %q", enginePipe)
	}

	// The policy registry key is written by the engine and read by the
	// installer during upgrade, so both sides must use the same key.
	policyKey := installerDefine(t, script, "HIMIX_CORE_POLICY_KEY")
	if !strings.Contains(enginePolicy, "`"+policyKey+"`") &&
		!strings.Contains(enginePolicy, strings.ReplaceAll(policyKey, "Software\\", "SOFTWARE\\")) {
		t.Fatalf("engine policy registry key does not match the installer (%q)", policyKey)
	}

	// No retired product name may survive in the engine's runtime strings.
	for name, source := range map[string]string{
		"service_policy_windows.go": enginePolicy,
		"service_windows.go":        engineService,
		"product_identity_windows.go": engine,
	} {
		for _, line := range strings.Split(source, "\n") {
			if !strings.Contains(line, "HypoMux") && !strings.Contains(line, "hypomux") {
				continue
			}
			// The Go module path still carries the repository name.
			if strings.Contains(line, "github.com/internest-nine/Himix/") {
				continue
			}
			t.Fatalf("%s still references the retired product name: %s", name, strings.TrimSpace(line))
		}
	}
}

func installerDefine(t *testing.T, script string, name string) string {
	t.Helper()
	pattern := regexp.MustCompile(`!define ` + regexp.QuoteMeta(name) + ` "([^"]*)"`)
	match := pattern.FindStringSubmatch(script)
	if match == nil {
		t.Fatalf("installer does not define %s", name)
	}
	return match[1]
}

func firstPipeName(t *testing.T, source string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?m)^[^\n]*PipeName[^\n]*= ` + "`" + `([^` + "`" + `]+)` + "`" + ``)
	match := pattern.FindStringSubmatch(source)
	if match == nil {
		t.Fatal("pipe name is not declared")
	}
	return match[1]
}
