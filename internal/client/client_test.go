// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommandLoginAndAPIPathFallback(t *testing.T) {
	t.Helper()
	var loginCount, getCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/api.cgi" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path != "/api.cgi" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		cmd, _ := body[0]["cmd"].(string)
		w.Header().Set("Content-Type", "application/json")
		switch cmd {
		case "Login":
			loginCount++
			_, _ = w.Write([]byte(`[{"cmd":"Login","code":0,"value":{"Token":{"name":"secret-token","leaseTime":3600}}}]`))
		case "GetAiAlarm":
			getCount++
			if got := r.URL.Query().Get("token"); got != "secret-token" {
				t.Errorf("token=%q", got)
			}
			_, _ = w.Write([]byte(`[{"cmd":"GetAiAlarm","code":0,"value":{"AiAlarm":{"sensitivity":60}}}]`))
		default:
			t.Errorf("unexpected command %q", cmd)
		}
	}))
	defer server.Close()
	c, err := New(Config{URL: server.URL, Username: "admin", Password: "camera-password"})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		e, err := c.Command(context.Background(), "GetAiAlarm", 1, map[string]any{"channel": 0, "ai_type": "people"})
		if err != nil {
			t.Fatal(err)
		}
		if e.Code != 0 {
			t.Fatalf("code=%d", e.Code)
		}
	}
	if loginCount != 1 {
		t.Fatalf("login count=%d, want 1", loginCount)
	}
	if getCount != 2 {
		t.Fatalf("get count=%d, want 2", getCount)
	}
}

func TestNewRejectsCredentialURL(t *testing.T) {
	_, err := New(Config{URL: "https://admin:secret@camera.example"})
	if err == nil {
		t.Fatal("expected credential-bearing URL to be rejected")
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") == "Login" {
			_, _ = w.Write([]byte(`[{"cmd":"Login","code":0,"value":{"Token":{"name":"t","leaseTime":3600}}}]`))
			return
		}
		_, _ = w.Write([]byte(`[{"cmd":"GetAiAlarm","code":1,"error":{"rspCode":-9,"detail":"ability not support"}}]`))
	}))
	defer server.Close()
	c, _ := New(Config{URL: server.URL, Username: "admin", Password: "secret"})
	_, err := c.Command(context.Background(), "GetAiAlarm", 1, map[string]any{})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestCommandRelogsAfterTokenExpires(t *testing.T) {
	var loginCount, commandCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/api.cgi" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("cmd") == "Login" {
			loginCount++
			_, _ = w.Write([]byte(`[{"cmd":"Login","code":0,"value":{"Token":{"name":"token","leaseTime":3600}}}]`))
			return
		}
		commandCount++
		if commandCount == 1 {
			_, _ = w.Write([]byte(`[{"cmd":"GetPushV20","code":1,"error":{"rspCode":-6,"detail":"please login first"}}]`))
			return
		}
		_, _ = w.Write([]byte(`[{"cmd":"GetPushV20","code":0,"value":{"Push":{"enable":0}}}]`))
	}))
	defer server.Close()

	c, err := New(Config{URL: server.URL, Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Command(context.Background(), "GetPushV20", 1, map[string]any{"channel": 0}); err != nil {
		t.Fatal(err)
	}
	if loginCount != 2 {
		t.Fatalf("login count=%d, want 2", loginCount)
	}
	if commandCount != 2 {
		t.Fatalf("command count=%d, want 2", commandCount)
	}
}
