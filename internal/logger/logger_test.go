package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewWritesRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := New("info", &buf)

	log.Info("handling request", FieldRequestID, "req-123")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v (%q)", err, buf.String())
	}
	if record[FieldRequestID] != "req-123" {
		t.Errorf("expected request_id req-123, got %v", record[FieldRequestID])
	}
	if record["msg"] != "handling request" {
		t.Errorf("expected msg, got %v", record["msg"])
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"error":   slog.LevelError,
		"unknown": slog.LevelInfo,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNewRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	log := New("error", &buf)

	log.Info("should be filtered")

	if buf.Len() != 0 {
		t.Errorf("info record emitted at error level: %q", buf.String())
	}
}
