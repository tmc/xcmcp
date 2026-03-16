package preview

import (
	"strings"
	"testing"
)

func TestInferViewType(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "plain struct",
			source: `
import SwiftUI

struct GreetingView: View {
	var body: some View { Text("hi") }
}
`,
			want: "GreetingView",
		},
		{
			name: "modifier and attribute",
			source: `
import SwiftUI

@MainActor
public struct ProfileCard: View {
	var body: some View { Text("profile") }
}
`,
			want: "ProfileCard",
		},
		{
			name: "no match",
			source: `
struct Model {
	let name: String
}
`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InferViewType(tt.source); got != tt.want {
				t.Fatalf("InferViewType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanGeneratedPreview(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text",
			in:   "#Preview {\n    GreetingView()\n}",
			want: "#Preview {\n    GreetingView()\n}",
		},
		{
			name: "fenced code",
			in:   "```swift\n#Preview {\n    GreetingView()\n}\n```",
			want: "#Preview {\n    GreetingView()\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanGeneratedPreview(tt.in); got != tt.want {
				t.Fatalf("CleanGeneratedPreview() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseVariants(t *testing.T) {
	got := ParseVariants("loading, error\nDark Mode, loading")
	want := []string{"loading", "error", "Dark Mode"}
	if len(got) != len(want) {
		t.Fatalf("len(ParseVariants()) = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseVariants()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildInstructionsIncludesVariants(t *testing.T) {
	got := BuildInstructions("GreetingView", []string{"loading", "error"}, "use sample data")
	if !strings.Contains(got, "loading, error") {
		t.Fatalf("BuildInstructions() = %q, want variants in instructions", got)
	}
	if !strings.Contains(got, "use sample data") {
		t.Fatalf("BuildInstructions() = %q, want notes in instructions", got)
	}
}
