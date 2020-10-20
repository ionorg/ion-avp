package avp

var registry *Registry

// Init avp with a registry of elements
func Init(elems map[string]ElementFun) {
	registry = NewRegistry()
	for eid, elem := range elems {
		registry.AddElement(eid, elem)
	}
}
