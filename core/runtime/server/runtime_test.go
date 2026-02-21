package server

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

func TestRuntimeLifecycle_StartReloadStop(t *testing.T) {
	port := freePort(t)
	model := &hyperterse.Model{Name: "runtime-lifecycle"}

	rt, err := NewRuntime(model, port, "test")
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	started := false
	if err := rt.StartAsync(); err != nil {
		t.Fatalf("StartAsync failed: %v", err)
	}
	started = true
	defer func() {
		if started {
			_ = rt.Stop()
		}
	}()

	heartbeatURL := fmt.Sprintf("http://127.0.0.1:%s/heartbeat", port)
	if err := waitForHTTP200(heartbeatURL, 5*time.Second); err != nil {
		t.Fatalf("heartbeat endpoint did not become healthy: %v", err)
	}

	// Ensure CORS wrapper remains active on /mcp.
	optionsReq, err := http.NewRequest(http.MethodOptions, fmt.Sprintf("http://127.0.0.1:%s/mcp", port), nil)
	if err != nil {
		t.Fatalf("failed to create /mcp OPTIONS request: %v", err)
	}
	optionsResp, err := http.DefaultClient.Do(optionsReq)
	if err != nil {
		t.Fatalf("failed to call /mcp OPTIONS: %v", err)
	}
	optionsResp.Body.Close()
	if optionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected /mcp OPTIONS status %d, got %d", http.StatusOK, optionsResp.StatusCode)
	}
	if optionsResp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin header on /mcp OPTIONS response")
	}
	if optionsResp.Header.Get("Access-Control-Expose-Headers") != "Mcp-Session-Id" {
		t.Fatalf("expected Access-Control-Expose-Headers to include Mcp-Session-Id on /mcp responses")
	}

	reloadedModel := &hyperterse.Model{Name: "runtime-lifecycle-reloaded"}
	if err := rt.ReloadModel(reloadedModel); err != nil {
		t.Fatalf("ReloadModel failed: %v", err)
	}
	if err := waitForHTTP200(heartbeatURL, 5*time.Second); err != nil {
		t.Fatalf("heartbeat endpoint not healthy after reload: %v", err)
	}

	if err := rt.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	started = false
}

func freePort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve free port: %v", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("failed to resolve reserved TCP address")
	}
	return fmt.Sprintf("%d", addr.Port)
}

func waitForHTTP200(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", url)
}
