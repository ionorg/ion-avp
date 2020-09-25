package avp

// Element interface
type Element interface {
	ID() string
	Write(*Sample) error
	Attach(Element) error
	Close()
}
