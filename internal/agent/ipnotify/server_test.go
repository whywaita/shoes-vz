package ipnotify

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestServer_StartStop(t *testing.T) {
	server := NewServer(0) // Use port 0 to get random available port

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestServer_WaitForIP(t *testing.T) {
	server := NewServer(18081)

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := server.Stop(context.Background()); err != nil {
			t.Logf("Stop() error = %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	runnerID := "test-runner-123"
	expectedIP := "192.168.64.5"

	// Send notification in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)

		notification := IPNotification{
			RunnerID:  runnerID,
			IPAddress: expectedIP,
		}

		body, _ := json.Marshal(notification)
		resp, err := http.Post("http://localhost:18081/notify-ip", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Errorf("Failed to send notification: %v", err)
			return
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}()

	// Wait for IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ip, err := server.WaitForIP(ctx, runnerID, 5*time.Second)
	if err != nil {
		t.Fatalf("WaitForIP() error = %v", err)
	}

	if ip != expectedIP {
		t.Errorf("WaitForIP() = %s, want %s", ip, expectedIP)
	}
}

func TestServer_WaitForIPTimeout(t *testing.T) {
	server := NewServer(18082)

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := server.Stop(context.Background()); err != nil {
			t.Logf("Stop() error = %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	_, err := server.WaitForIP(ctx, "non-existent-runner", 500*time.Millisecond)
	if err == nil {
		t.Error("WaitForIP() expected timeout error, got nil")
	}
}

func TestServer_HandleIPNotification_InvalidMethod(t *testing.T) {
	server := NewServer(18083)

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := server.Stop(context.Background()); err != nil {
			t.Logf("Stop() error = %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:18083/notify-ip")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestServer_HandleIPNotification_InvalidJSON(t *testing.T) {
	server := NewServer(18084)

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := server.Stop(context.Background()); err != nil {
			t.Logf("Stop() error = %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Post("http://localhost:18084/notify-ip", "application/json", bytes.NewReader([]byte("invalid json")))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestServer_HandleIPNotification_MissingFields(t *testing.T) {
	server := NewServer(18085)

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := server.Stop(context.Background()); err != nil {
			t.Logf("Stop() error = %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name         string
		notification IPNotification
	}{
		{
			name: "missing runner_id",
			notification: IPNotification{
				IPAddress: "192.168.64.5",
			},
		},
		{
			name: "missing ip_address",
			notification: IPNotification{
				RunnerID: "test-runner",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.notification)
			resp, err := http.Post("http://localhost:18085/notify-ip", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", resp.StatusCode)
			}
		})
	}
}
