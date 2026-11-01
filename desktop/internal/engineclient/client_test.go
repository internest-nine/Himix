package engineclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRealEngineHandshakeAndStoppedStatus(t *testing.T) {
	path := os.Getenv("HIMIX_ENGINE_PATH")
	if path == "" {
		path = filepath.Clean(filepath.Join("..", "..", "..", "..", "himix-engine.exe"))
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("real himix-engine.exe is not available")
	}
	t.Setenv("HIMIX_ENGINE_PATH", path)
	client := New()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	hello, err := client.Ensure(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown(ctx)
	if hello.ProtocolVersion != ProtocolVersion {
		t.Fatalf("unexpected protocol %d", hello.ProtocolVersion)
	}
	var status struct {
		Engine struct {
			State string `json:"state"`
		} `json:"engine"`
	}
	if err := client.Request(ctx, "engine.status", nil, &status); err != nil {
		t.Fatal(err)
	}
	if status.Engine.State != "stopped" {
		t.Fatalf("expected stopped engine, got %q", status.Engine.State)
	}
}
