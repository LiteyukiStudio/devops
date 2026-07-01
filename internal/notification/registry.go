package notification

import (
	"fmt"
	"strings"
)

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) Registry {
	registry := Registry{adapters: map[string]Adapter{}}
	for _, adapter := range adapters {
		registry.Register(adapter)
	}
	return registry
}

func DefaultRegistry() Registry {
	return NewRegistry(WebhookAdapter{}, SMTPAdapter{})
}

func (r *Registry) Register(adapter Adapter) {
	if adapter == nil {
		return
	}
	kind := strings.TrimSpace(adapter.Kind())
	if kind == "" {
		return
	}
	if r.adapters == nil {
		r.adapters = map[string]Adapter{}
	}
	r.adapters[kind] = adapter
}

func (r Registry) Adapter(kind string) (Adapter, error) {
	adapter, ok := r.adapters[strings.TrimSpace(kind)]
	if !ok {
		return nil, fmt.Errorf("notification adapter %q is not registered", kind)
	}
	return adapter, nil
}
