package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The raw CDN that normally serves update-channel files is unreachable on some
// networks, so the same signed manifest is also fetched through the GitHub git
// data API. That transport has to return the manifest and its binary signature
// byte for byte; the contents endpoint re-encodes binary payloads and would
// break verification.
func TestUpdateManifestIsFetchedThroughTheGitDataAPI(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"schema_version":1,"version":"0.3","name":"Himix 0.3","notes":"n","installer":{"name":"Himix_Setup_0.3.exe","size":123,"sha256":"` + strings.Repeat("a", 64) + `","urls":["https://github.com/internest-nine/Himix/releases/download/v0.3/Himix_Setup_0.3.exe"]}}`)
	signature := ed25519.Sign(private, manifest)

	blobs := map[string][]byte{
		"manifest-sha":  manifest,
		"signature-sha": signature,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/git/trees/update-channel":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tree":[{"path":"latest.json","sha":"manifest-sha"},{"path":"latest.json.sig","sha":"signature-sha"}]}`)
		case strings.HasPrefix(r.URL.Path, "/git/blobs/"):
			sha := strings.TrimPrefix(r.URL.Path, "/git/blobs/")
			payload, ok := blobs[sha]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// The real endpoint returns raw bytes for this Accept value.
			if r.Header.Get("Accept") != "application/vnd.github.raw" {
				t.Errorf("blob read did not ask for raw bytes: %q", r.Header.Get("Accept"))
			}
			w.Write(payload)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	service := NewUpdaterService()
	service.manifestPublicKey = public
	service.updateAPIBaseURL = server.URL

	release, err := service.checkUpdateManifestThroughAPI(context.Background(), stableChannelRef)
	if err != nil {
		t.Fatalf("API transport failed: %v", err)
	}
	if release.TagName != "v0.3" {
		t.Fatalf("unexpected tag %q", release.TagName)
	}
	if release.InstallerName != "Himix_Setup_0.3.exe" {
		t.Fatalf("unexpected installer %q", release.InstallerName)
	}
}

func TestUpdateManifestThroughAPIRejectsABrokenSignature(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"schema_version":1,"version":"0.4","name":"Himix 0.4","notes":"n","installer":{"name":"Himix_Setup_0.4.exe","size":1,"sha256":"` + strings.Repeat("b", 64) + `","urls":["https://github.com/internest-nine/Himix/releases/download/v0.4/Himix_Setup_0.4.exe"]}}`)
	signature := ed25519.Sign(private, manifest)
	signature[0] ^= 0xff // one flipped byte is enough

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/git/trees/update-channel":
			fmt.Fprint(w, `{"tree":[{"path":"latest.json","sha":"m"},{"path":"latest.json.sig","sha":"s"}]}`)
		case r.URL.Path == "/git/blobs/m":
			w.Write(manifest)
		case r.URL.Path == "/git/blobs/s":
			w.Write(signature)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	service := NewUpdaterService()
	service.manifestPublicKey = public
	service.updateAPIBaseURL = server.URL

	if _, err := service.checkUpdateManifestThroughAPI(context.Background(), stableChannelRef); err == nil {
		t.Fatal("a corrupted signature was accepted")
	}
}

func TestUpdateManifestThroughAPIOnlyAcceptsKnownChannels(t *testing.T) {
	service := NewUpdaterService()
	for _, ref := range []string{"main", "update-channel/../../evil", ""} {
		if _, err := service.checkUpdateManifestThroughAPI(context.Background(), ref); err == nil {
			t.Fatalf("channel ref %q was accepted", ref)
		}
	}
}

// Guard the constants the API transport depends on: they must describe the
// same repository the direct URLs point at.
func TestUpdateChannelConstantsAgreeOnRepository(t *testing.T) {
	if !strings.Contains(githubAPIRepositoryURL, "internest-nine/Himix") {
		t.Fatalf("API base points somewhere else: %s", githubAPIRepositoryURL)
	}
	if !strings.Contains(githubLatestManifestURL, stableChannelRef) {
		t.Fatalf("stable manifest URL does not use %q", stableChannelRef)
	}
	if !strings.Contains(githubPreviewManifestURL, previewChannelRef) {
		t.Fatalf("preview manifest URL does not use %q", previewChannelRef)
	}
}
