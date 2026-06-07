package main

import (
	"strings"
	"testing"
)

func TestRunRequiresCommand(t *testing.T) {
	err := run(nil)
	if err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("err = %v", err)
	}
}
