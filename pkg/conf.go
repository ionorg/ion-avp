package avp

type Samplebuilderconf struct {
	AudioMaxLate  uint16 `mapstructure:"audiomaxlate"`
	VideoMaxLate  uint16 `mapstructure:"videomaxlate"`
	MaxLateTimeMs uint32 `mapstructure:"maxlatems"`
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

// Config defines parameters for the logger
type logConf struct {
	Level string `mapstructure:"level"`
}

// Config for base AVP
type Config struct {
	Log           logConf           `mapstructure:"log"`
	SampleBuilder Samplebuilderconf `mapstructure:"samplebuilder"`
	WebRTC        webrtcconf        `mapstructure:"webrtc"`
}
