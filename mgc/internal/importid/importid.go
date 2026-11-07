package importid

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Namespace struct {
	Name     string
	Validate func(string) error
}

type Parsed struct {
	ID     string
	values map[string]string
}

func (p Parsed) Value(ns Namespace) types.String {
	if v, ok := p.values[ns.Name]; ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

// resourceKey names the pair that carries the resource id; it always closes
// the import ID, after any namespace pairs.
const resourceKey = "resource"

func Parse(raw string, accepted ...Namespace) (Parsed, error) {
	segments := strings.Split(raw, "/")
	if len(segments) == 1 {
		segments = []string{resourceKey, raw}
	}

	last := len(segments) - 2
	if len(segments)%2 != 0 || segments[last] != resourceKey {
		return Parsed{}, fmt.Errorf("invalid import ID %q: expected {id} or namespace/value pairs ending in %s/{id}, accepted namespaces: %s", raw, resourceKey, names(accepted))
	}
	id, pairs := segments[last+1], segments[:last]
	if id == "" {
		return Parsed{}, fmt.Errorf("invalid import ID %q: empty resource id", raw)
	}

	parsed := Parsed{ID: id, values: map[string]string{}}
	for i := 0; i < len(pairs); i += 2 {
		name, value := pairs[i], pairs[i+1]

		if name == resourceKey {
			return Parsed{}, fmt.Errorf("invalid import ID %q: %q must be the last pair", raw, resourceKey)
		}
		ns, ok := find(accepted, name)
		if !ok {
			return Parsed{}, fmt.Errorf("invalid import ID %q: unknown namespace %q, accepted namespaces: %s", raw, name, names(accepted))
		}
		if _, repeated := parsed.values[name]; repeated {
			return Parsed{}, fmt.Errorf("invalid import ID %q: namespace %q repeated", raw, name)
		}
		if value == "" {
			return Parsed{}, fmt.Errorf("invalid import ID %q: namespace %q has no value", raw, name)
		}
		if ns.Validate != nil {
			if err := ns.Validate(value); err != nil {
				return Parsed{}, fmt.Errorf("invalid import ID %q: %w", raw, err)
			}
		}

		parsed.values[name] = value
	}

	return parsed, nil
}

func find(accepted []Namespace, name string) (Namespace, bool) {
	for _, ns := range accepted {
		if ns.Name == name {
			return ns, true
		}
	}
	return Namespace{}, false
}

func names(accepted []Namespace) string {
	if len(accepted) == 0 {
		return "none"
	}
	out := make([]string, len(accepted))
	for i, ns := range accepted {
		out[i] = ns.Name
	}
	return strings.Join(out, ", ")
}
