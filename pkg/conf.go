package avp

import log "github.com/pion/ion-log"

type Samplebuilderconf struct {
	AudioMaxLate uint16 `mapstructure:"audiomaxlate"`
	VideoMaxLate uint16 `mapstructure:"videomaxlate"`
}

type iceconf struct {
	URLs       []string `mapstructure:"urls"`
	Username   string   `mapstructure:"username"`
	Credential string   `mapstructure:"credential"`
}

type webrtcconf struct {
	PLICycle     uint      `mapstructure:"plicycle"`
	ICEPortRange []uint16  `mapstructure:"portrange"`
	ICEServers   []iceconf `mapstructure:"iceserver"`
}

// Config for base AVP
type Config struct {
	Log           log.Config        `mapstructure:"log"`
	SampleBuilder Samplebuilderconf `mapstructure:"samplebuilder"`
	WebRTC        webrtcconf        `mapstructure:"webrtc"`
}
