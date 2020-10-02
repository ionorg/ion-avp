package avp

// Types for samples
const (
	TypeOpus = 1
	TypeVP8  = 2
	TypeVP9  = 3
	TypeH264 = 4
)

// Sample of audio or video
type Sample struct {
	Type           int
	Timestamp      uint32
	SequenceNumber uint16
	Payload        interface{}
	TrackID        string
}
