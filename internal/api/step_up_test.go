package api

import "testing"

func TestConfigBool(t *testing.T) {
	for _, value := range []string{"true", "1", "yes", "on", "enabled", " TRUE "} {
		if !configBool(value) {
			t.Fatalf("configBool(%q) = false", value)
		}
	}
	for _, value := range []string{"", "false", "0", "no", "off", "disabled"} {
		if configBool(value) {
			t.Fatalf("configBool(%q) = true", value)
		}
	}
}
