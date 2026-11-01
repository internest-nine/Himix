package services

import (
	"github.com/internest-nine/Himix/desktop/internal/engineclient"
	"testing"
)

func TestLatencySchedulingPersistence(t *testing.T) {
	t.Setenv("HIMIX_DATA_DIR", t.TempDir())
	settings := NewSettingsService()
	s := &EngineService{settings: settings, adapters: NewAdapterService(settings), client: engineclient.New(), lifecycleGate: make(chan struct{}, 1)}
	for _, mode := range []string{"proxy", "tun"} {
		if _, err := s.SaveScheduling(mode, "latency-first", nil); err != nil {
			t.Fatal(err)
		}
		loaded := NewSettingsService().Get()
		if loaded.Strategy != "latency-first" || loaded.Weighted || loaded.Mode != mode {
			t.Fatal(loaded)
		}
	}
}
