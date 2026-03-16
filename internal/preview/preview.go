package preview

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var swiftUIViewRE = regexp.MustCompile(`(?m)^\s*(?:@\w+(?:\([^)]*\))?\s*)*(?:(?:public|internal|fileprivate|private|final)\s+)*struct\s+([A-Za-z_][A-Za-z0-9_]*)\s*:\s*[^\n{]*\bView\b`)

type Spec struct {
	Path         string
	TypeName     string
	Variants     []string
	Notes        string
	Source       string
	SourceURI    string
	SourceMIME   string
	Instructions string
}

func Prepare(path, typeName string, variants []string, notes string) (*Spec, error) {
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	if filepath.Ext(absPath) != ".swift" {
		return nil, fmt.Errorf("path must point to a .swift file")
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read source file: %w", err)
	}

	source := string(data)
	if typeName == "" {
		typeName = InferViewType(source)
	}
	if typeName == "" {
		return nil, fmt.Errorf("type_name is required when no SwiftUI view can be inferred from %s", filepath.Base(absPath))
	}

	variants = NormalizeVariants(variants)
	notes = strings.TrimSpace(notes)
	return &Spec{
		Path:         absPath,
		TypeName:     typeName,
		Variants:     variants,
		Notes:        notes,
		Source:       source,
		SourceURI:    SourceURI(absPath),
		SourceMIME:   "text/x-swift",
		Instructions: BuildInstructions(typeName, variants, notes),
	}, nil
}

func InferViewType(source string) string {
	m := swiftUIViewRE.FindStringSubmatch(source)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func BuildInstructions(typeName string, variants []string, notes string) string {
	var b strings.Builder
	b.WriteString("Generate a SwiftUI #Preview block for ")
	b.WriteString(typeName)
	b.WriteString(". Return only Swift code for the preview block.")
	b.WriteString(" Do not include markdown fences or explanation.")
	b.WriteString(" Do not rewrite the existing view implementation.")
	b.WriteString(" Prefer modern #Preview syntax over PreviewProvider.")
	if len(variants) > 0 {
		b.WriteString(" Cover these variants with separate named previews when appropriate: ")
		b.WriteString(strings.Join(variants, ", "))
		b.WriteString(".")
	}
	if notes != "" {
		b.WriteString(" Additional requirements: ")
		b.WriteString(notes)
	}
	return b.String()
}

func ParseVariants(text string) []string {
	text = strings.ReplaceAll(text, "\n", ",")
	return NormalizeVariants(strings.Split(text, ","))
}

func NormalizeVariants(variants []string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, variant := range variants {
		variant = strings.TrimSpace(variant)
		if variant == "" {
			continue
		}
		key := strings.ToLower(variant)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, variant)
	}
	return out
}

func SourceURI(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func CleanGeneratedPreview(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	lines = lines[1:]
	if n := len(lines); n > 0 && strings.HasPrefix(strings.TrimSpace(lines[n-1]), "```") {
		lines = lines[:n-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
