package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The About page points at files under public/support, which Wails embeds into
// the binary at build time. A missing or renamed file produces two broken
// images in a shipped build with no error at any stage, so every reference is
// checked against the filesystem here.
func TestFrontendSupportAssetsExist(t *testing.T) {
	reference := regexp.MustCompile(`["']/support/([^"']+)["']`)

	root := filepath.Join("frontend", "src")
	referenced := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".tsx") {
			return nil
		}
		// Test files describe their own mocks and never ship in the binary.
		if strings.HasSuffix(path, ".test.tsx") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range reference.FindAllStringSubmatch(string(content), -1) {
			referenced[match[1]] = path
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scanning frontend sources: %v", err)
	}

	if len(referenced) == 0 {
		t.Fatal("found no /support/ references; the scan is not reading the sources")
	}

	for asset, source := range referenced {
		target := filepath.Join("frontend", "public", "support", filepath.FromSlash(asset))
		info, statErr := os.Stat(target)
		if statErr != nil {
			t.Errorf("%s references /support/%s but %s is missing", source, asset, target)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s references /support/%s but the file is empty", source, asset)
		}
	}
}

// The tip codes are the assets most likely to be forgotten, because they are
// added by hand rather than generated. Keep an explicit expectation so the
// generic scan above cannot pass by finding nothing.
func TestTipCodeAssetsArePresent(t *testing.T) {
	for _, name := range []string{"qr-wechat.png", "qr-alipay.png"} {
		path := filepath.Join("frontend", "public", "support", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s is missing: %v", path, err)
		}
		// A real QR image is tens of kilobytes; anything tiny is a placeholder.
		if info.Size() < 4096 {
			t.Fatalf("%s looks like a placeholder (%d bytes)", path, info.Size())
		}
	}
}
