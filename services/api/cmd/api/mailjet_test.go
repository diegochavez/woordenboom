package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendMailjetLoginCode(t *testing.T) {
	var received bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != http.MethodPost || r.URL.Path != "/v3.1/send" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		key, secret, ok := r.BasicAuth()
		if !ok || key != "api-key" || secret != "secret-key" {
			t.Error("missing Mailjet Basic authentication")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected JSON request")
		}
		var payload struct {
			Messages []mailjetMessage `json:"Messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(payload.Messages) != 1 {
			t.Errorf("expected one message, got %d", len(payload.Messages))
		} else {
			message := payload.Messages[0]
			if message.From.Email != "login@woordenboom.app" || len(message.To) != 1 || message.To[0].Email != "reader@example.com" || !strings.Contains(message.TextPart, "123456") {
				t.Errorf("unexpected Mailjet payload: %+v", message)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Messages":[{"Status":"success"}]}`))
	}))
	defer server.Close()

	if err := sendMailjetLoginCode(context.Background(), server.Client(), server.URL+"/v3.1/send", "api-key", "secret-key", "login@woordenboom.app", "reader@example.com", "123456"); err != nil {
		t.Fatalf("send login code: %v", err)
	}
	if !received {
		t.Fatal("Mailjet endpoint was not called")
	}
}

func TestSendMailjetLoginCodeRejectsFailedResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", http.StatusUnauthorized, `{"ErrorMessage":"secret data"}`},
		{"message error", http.StatusOK, `{"Messages":[{"Status":"error"}]}`},
		{"invalid response", http.StatusOK, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			err := sendMailjetLoginCode(context.Background(), server.Client(), server.URL, "api-key", "secret-key", "login@woordenboom.app", "reader@example.com", "123456")
			if err == nil {
				t.Fatal("expected Mailjet response to fail")
			}
			if strings.Contains(err.Error(), "secret data") || strings.Contains(err.Error(), "123456") {
				t.Fatal("Mailjet error exposed sensitive data")
			}
		})
	}
}
