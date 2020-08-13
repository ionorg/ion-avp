package avp

import "github.com/pion/ion-avp/pkg/log"

type samplebuilder struct {
	AudioMaxLate uint16 `mapstructure:"audiomaxlate"`
	VideoMaxLate uint16 `mapstructure:"videomaxlate"`
}

type pipeline struct {
	WebmSaver webmsaver `mapstructure:"webmsaver"`
}

type webmsaver struct {
	Path string `mapstructure:"path"`
}

type iceconf struct {
	URLs       []string `mapstructure:"urls"`
	Username   string   `mapstructure:"username"`
	Credential string   `mapstructure:"credential"`
}

type webrtcconf struct {
	ICEPortRange []uint16  `mapstructure:"portrange"`
	ICEServers   []iceconf `mapstructure:"iceserver"`
}

// Config for base AVP
type Config struct {
	Log           log.Config    `mapstructure:"log"`
	Pipeline      pipeline      `mapstructure:"pipeline"`
	SampleBuilder samplebuilder `mapstructure:"samplebuilder"`
	WebRTC        webrtcconf    `mapstructure:"webrtc"`
}
