package avp

// Registry provides a registry of elements
type Registry struct {
	elements map[string]func(id string) Element
}

// NewRegistry returns new registry instance
func NewRegistry() *Registry {
	return &Registry{
		elements: make(map[string]func(id string) Element),
	}
}

// AddElement to registry
func (r *Registry) AddElement(id string, f func(id string) Element) {
	r.elements[id] = f
}

// GetElement to registry
func (r *Registry) GetElement(id string) func(string) Element {
	return r.elements[id]
}
