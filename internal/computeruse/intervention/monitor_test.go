package intervention

import (
	"testing"
	"time"
)

func TestMonitorDisabledDoesNotBlock(t *testing.T) {
	m := New(Config{})
	now := time.Unix(10, 0)
	m.Record("KCGEventKeyDown", now)
	if _, blocked := m.Blocked(now); blocked {
		t.Fatalf("disabled monitor blocked action")
	}
}

func TestMonitorBlocksDuringQuietPeriod(t *testing.T) {
	m := New(Config{Enabled: true, QuietPeriod: time.Second})
	now := time.Unix(10, 0)
	m.Record("KCGEventKeyDown", now)

	status, blocked := m.Blocked(now.Add(500 * time.Millisecond))
	if !blocked {
		t.Fatalf("Blocked = false, want true")
	}
	if status.LastType != "KCGEventKeyDown" {
		t.Fatalf("LastType = %q, want KCGEventKeyDown", status.LastType)
	}
	if _, blocked := m.Blocked(now.Add(2 * time.Second)); blocked {
		t.Fatalf("Blocked after quiet period = true, want false")
	}
}
