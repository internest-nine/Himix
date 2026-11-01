package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The updater only accepts installers whose name matches installerNamePattern.
// The release workflow is the only producer of the published installer file, so
// the two must agree on the exact same name. Renaming the product without
// updating the workflow silently ships an installer that no client can update
// from, which is why this contract is asserted from both sides.
func TestPublishedInstallerNameMatchesUpdaterPattern(t *testing.T) {
	updaterData, err := os.ReadFile("internal/services/updater.go")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`installerNamePattern = regexp\.MustCompile\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	match := pattern.FindStringSubmatch(string(updaterData))
	if match == nil {
		t.Fatal("updater does not declare installerNamePattern")
	}
	accepted := match[1]

	workflowData, err := os.ReadFile("../.github/workflows/build.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(workflowData)

	// The workflow stages the installer under a versioned name; recover the
	// literal prefix it uses so we can compare it against the updater regex.
	staged := regexp.MustCompile(`cp artifacts/[^ ]+ "artifacts/([A-Za-z0-9_]+)_\$\{version\}\.exe"`)
	stagedMatch := staged.FindStringSubmatch(workflow)
	if stagedMatch == nil {
		t.Fatal("release workflow does not stage a versioned installer name")
	}
	prefix := stagedMatch[1]

	sample := prefix + "_1.2.3.exe"
	acceptedPattern := regexp.MustCompile(accepted)
	if !acceptedPattern.MatchString(sample) {
		t.Fatalf("release workflow publishes %q but the updater only accepts %q", sample, accepted)
	}
	if !strings.Contains(workflow, `installer_name="`+sample[:len(sample)-len("_1.2.3.exe")]+`_${version}.exe"`) {
		t.Fatal("release workflow stages and publishes inconsistent installer names")
	}
}

// The CI workflows must reference the product's real build outputs. Renaming
// the binaries without updating the workflows leaves the pipelines building
// paths that no longer exist.
func TestWorkflowsReferenceExistingBuildOutputs(t *testing.T) {
	workflowData, err := os.ReadFile("../.github/workflows/build.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(workflowData)

	for _, required := range []string{
		`INSTALLER_PATH: desktop/bin/himix-amd64-installer.exe`,
		`desktop/bin/himix.exe`,
		`desktop/bin/himix-engine.exe`,
		`Copy-Item desktop\bin\himix.exe signing-input\himix-unsigned.exe -Force`,
		`Copy-Item desktop\bin\himix-engine.exe signing-input\himix-engine-unsigned.exe -Force`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("release workflow does not reference the real build output: missing %q", required)
		}
	}

	// The helper test env var is read by the Go test suite; a mismatch makes
	// the test silently skip instead of actually running.
	if !strings.Contains(workflow, `HIMIX_DESKTOP_HELPER_TEST`) {
		t.Fatal("release workflow does not set HIMIX_DESKTOP_HELPER_TEST for the native helper test")
	}
	if !strings.Contains(workflow, `HIMIX_SIGNED_INSTALLER_TEST`) {
		t.Fatal("release workflow does not set HIMIX_SIGNED_INSTALLER_TEST for the installer trust test")
	}

	engineData, err := os.ReadFile("../.github/workflows/go-engine.yml")
	if err != nil {
		t.Fatal(err)
	}
	engine := string(engineData)
	if !strings.Contains(engine, `.\cmd\himix-engine`) {
		t.Fatal("engine workflow does not build the renamed engine command")
	}
	if strings.Contains(engine, `hypomux-engine`) {
		t.Fatal("engine workflow still references the retired hypomux-engine command")
	}
}
