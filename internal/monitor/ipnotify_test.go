package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNotifyIP(t *testing.T) {
	// Create a test server that accepts IP notifications
	receivedNotification := make(chan map[string]string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var notification map[string]string
		if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedNotification <- notification
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Skip if we can't get own IP (e.g., no 192.168.64.x interface)
	_, err := getOwnIP()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Note: NotifyIP uses getOwnIP() which requires a real 192.168.64.x interface
	// This test will be skipped if running outside a VM with that network configuration
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Extract host and port from test server URL
	// server.URL is like "http://127.0.0.1:12345"
	// We need to pass the host and port separately
	// For this test, we'll just verify the error handling when no 192.168.64.x IP exists

	// Test with invalid context (already cancelled)
	cancelledCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc() // Cancel immediately

	err = NotifyIP(cancelledCtx, "test-runner", "127.0.0.1", 8081)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}

	// Test with valid context but unreachable server (will retry until timeout)
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer shortCancel()

	err = NotifyIP(shortCtx, "test-runner", "127.0.0.1", 9999)
	if err == nil {
		t.Error("Expected error with unreachable server, got nil")
	}

	// We can't easily test the success case without mocking getOwnIP
	// or running in an actual VM environment with 192.168.64.x network
	_ = ctx
}

func TestGetOwnIP(t *testing.T) {
	// This test will only pass if running in a VM with 192.168.64.x network
	ip, err := getOwnIP()
	if err != nil {
		t.Skipf("Skipping test: no 192.168.64.x IP found (expected if not in VM): %v", err)
	}

	// Verify IP format if found
	if len(ip) == 0 {
		t.Error("getOwnIP() returned empty IP")
	}

	// IP should start with 192.168.64
	if len(ip) < 12 || ip[:11] != "192.168.64." {
		t.Errorf("getOwnIP() = %s, expected 192.168.64.x", ip)
	}
}
