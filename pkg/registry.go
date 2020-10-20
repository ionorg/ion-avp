package avp

// ElementFun create a element
type ElementFun func(sid, pid, tid string, config []byte) Element

// Registry provides a registry of elements
type Registry struct {
	elements map[string]ElementFun
}

// NewRegistry returns new registry instance
func NewRegistry() *Registry {
	return &Registry{
		elements: make(map[string]ElementFun),
	}
}

// AddElement to registry
func (r *Registry) AddElement(eid string, f ElementFun) {
	r.elements[eid] = f

}

// GetElement to registry
func (r *Registry) GetElement(id string) ElementFun {
	return r.elements[id]
}
