package kubernetes

import (
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
)

func TestDataExportPodNameIsUniqueAndDNS1123SafeUnderConcurrency(t *testing.T) {
	const count = 256
	names := make(chan string, count)
	errors := make(chan error, count)
	var waitGroup sync.WaitGroup
	for range count {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			name, err := dataExportPodName("DATA_export_target_with_a_very_long_name_that_would_otherwise_exceed_the_dns_label_limit")
			if err != nil {
				errors <- err
				return
			}
			names <- name
		}()
	}
	waitGroup.Wait()
	close(names)
	close(errors)
	for err := range errors {
		t.Fatalf("generate pod name: %v", err)
	}

	seen := make(map[string]struct{}, count)
	for name := range names {
		if problems := validation.IsDNS1123Label(name); len(problems) > 0 {
			t.Fatalf("pod name %q is invalid: %v", name, problems)
		}
		if len(name) > 63 {
			t.Fatalf("pod name %q exceeds 63 characters", name)
		}
		if _, duplicate := seen[name]; duplicate {
			t.Fatalf("duplicate pod name generated: %q", name)
		}
		seen[name] = struct{}{}
	}
	if len(seen) != count {
		t.Fatalf("generated %d unique names, want %d", len(seen), count)
	}
}
