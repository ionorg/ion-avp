package avp

import "github.com/pion/ion-avp/pkg/log"

type grpc struct {
	Port string `mapstructure:"port"`
}

type samplebuilder struct {
	AudioMaxLate uint16 `mapstructure:"audiomaxlate"`
	VideoMaxLate uint16 `mapstructure:"videomaxlate"`
}

type pipeline struct {
	WebmSaver webmsaver `mapstructure:"webmsaver"`
}

type webmsaver struct {
	Enabled   bool   `mapstructure:"enabled"`
	Togglable bool   `mapstructure:"togglable"`
	DefaultOn bool   `mapstructure:"defaulton"`
	Path      string `mapstructure:"path"`
}

// Config for base AVP
type Config struct {
	GRPC          grpc          `mapstructure:"grpc"`
	Pipeline      pipeline      `mapstructure:"pipeline"`
	Log           log.Config    `mapstructure:"log"`
	SampleBuilder samplebuilder `mapstructure:"samplebuilder"`
}
