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

// Config for base AVP
type Config struct {
	Pipeline      pipeline      `mapstructure:"pipeline"`
	Log           log.Config    `mapstructure:"log"`
	SampleBuilder samplebuilder `mapstructure:"samplebuilder"`
}
