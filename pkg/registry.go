package avp

// Registry provides a registry of elements
type Registry struct {
	elements map[string]func(sid, pid, tid string) Element
}

// NewRegistry returns new registry instance
func NewRegistry() *Registry {
	return &Registry{
		elements: make(map[string]func(sid, pid, tid string) Element),
	}
}

// AddElement to registry
func (r *Registry) AddElement(eid string, f func(sid, pid, tid string) Element) {
	r.elements[eid] = f
}

// GetElement to registry
func (r *Registry) GetElement(id string) func(string, string, string) Element {
	return r.elements[id]
}
