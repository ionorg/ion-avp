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
	SampleBuilder samplebuilder `mapstructure:"samplebuilder"`
	WebmSaver     webmsaver     `mapstructure:"webmsaver"`
}

type webmsaver struct {
	Enabled   bool   `mapstructure:"enabled"`
	Togglable bool   `mapstructure:"togglable"`
	DefaultOn bool   `mapstructure:"defaulton"`
	Path      string `mapstructure:"path"`
}

type rtp struct {
	Port    int    `mapstructure:"port"`
	KcpKey  string `mapstructure:"kcpkey"`
	KcpSalt string `mapstructure:"kcpsalt"`
}

// Config for base AVP
type Config struct {
	GRPC     grpc       `mapstructure:"grpc"`
	Pipeline pipeline   `mapstructure:"pipeline"`
	Rtp      rtp        `mapstructure:"rtp"`
	Log      log.Config `mapstructure:"log"`
	CfgFile  string
}
