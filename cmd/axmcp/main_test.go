package main

import (
	"strings"
	"testing"
	"time"
)

func TestPermissionPane(t *testing.T) {
	tests := []struct {
		service string
		want    string
	}{
		{service: "Accessibility", want: "Accessibility"},
		{service: "Screen Recording", want: "Screen Recording"},
	}

	for _, tt := range tests {
		if got := permissionPane(tt.service); got != tt.want {
			t.Fatalf("permissionPane(%q) = %q, want %q", tt.service, got, tt.want)
		}
	}
}

func TestWaitForPermissionImmediate(t *testing.T) {
	if err := waitForPermission("Accessibility", time.Millisecond, time.Microsecond, func() bool { return true }); err != nil {
		t.Fatalf("waitForPermission returned %v, want nil", err)
	}
}

func TestWaitForPermissionTimeout(t *testing.T) {
	err := waitForPermission("Accessibility", 5*time.Millisecond, time.Millisecond, func() bool { return false })
	if err == nil {
		t.Fatal("waitForPermission returned nil, want error")
	}
	if !strings.Contains(err.Error(), "Accessibility permission not granted for axmcp.app") {
		t.Fatalf("waitForPermission error = %q, want accessibility guidance", err)
	}
}
