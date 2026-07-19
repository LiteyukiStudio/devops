package buildtemplate

import (
	"strings"
	"testing"
)

func TestBuiltInTemplatesRenderWithDefaults(t *testing.T) {
	for _, definition := range List() {
		t.Run(definition.ID, func(t *testing.T) {
			preview, err := Render(definition.ID, definition.Version, nil)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if !strings.HasPrefix(preview.Dockerfile, "FROM ") {
				t.Fatalf("Dockerfile = %q", preview.Dockerfile)
			}
			if strings.Contains(preview.Dockerfile, "{{") {
				t.Fatalf("Dockerfile contains an unresolved template expression: %q", preview.Dockerfile)
			}
			if len(preview.Checksum) != 64 {
				t.Fatalf("checksum length = %d", len(preview.Checksum))
			}
		})
	}
}

func TestRenderEscapesRuntimeCommandAsJSON(t *testing.T) {
	preview, err := Render("node-service", "", map[string]string{
		"startCommand": `node -e "console.log('ready')"`,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(preview.Dockerfile, `CMD ["sh", "-c", "node -e \"console.log('ready')\""]`) {
		t.Fatalf("runtime command was not JSON escaped: %s", preview.Dockerfile)
	}
}

func TestNormalizeValuesRejectsUnsafeOrUnknownValues(t *testing.T) {
	definition, ok := Find("static-site", "")
	if !ok {
		t.Fatal("static-site template not found")
	}
	for name, raw := range map[string]string{
		"parent path":   `{"sourceDirectory":"../private"}`,
		"absolute path": `{"sourceDirectory":"/private"}`,
		"unknown key":   `{"extra":"value"}`,
		"newline":       `{"sourceDirectory":"public\\nRUN whoami"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeValues(definition, raw); err == nil {
				t.Fatal("NormalizeValues() error = nil")
			}
		})
	}
}

func TestRenderRejectsPortWithTrailingCharacters(t *testing.T) {
	if _, err := Render("go-service", "", map[string]string{"port": "8080abc"}); err == nil {
		t.Fatal("Render() accepted a port with trailing characters")
	}
}

func TestRecommendPrefersMoreSpecificTemplate(t *testing.T) {
	got := Recommend([]string{"package.json", "vite.config.ts", "src/main.ts"})
	if len(got) < 2 || got[0] != "node-static" || got[1] != "node-service" {
		t.Fatalf("Recommend() = %#v", got)
	}
}
