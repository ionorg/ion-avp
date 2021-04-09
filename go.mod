module github.com/pion/ion-avp

go 1.15

require (
	github.com/at-wat/ebml-go v0.14.1
	github.com/golang/protobuf v1.4.3
	github.com/lucsky/cuid v1.0.2
	github.com/pion/ion-log v1.0.0
	github.com/pion/ion-sfu v1.9.8-0.20210409201849-5750f5de326a // go get -u github.com/pion/ion-sfu@robin-20210119-seqnum
	github.com/pion/rtcp v1.2.6
	github.com/pion/rtp v1.6.3-0.20210128035234-5b3f2454a01a // go get -u github.com/pion/rtp@robin-20210119-seqnum
	github.com/pion/transport v0.12.3
	github.com/pion/webrtc/v3 v3.0.22-0.20210409200319-167a194a5698 // go get -u github.com/pion/webrtc/v3@robin-20210119-seqnum
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/xlab/libvpx-go v0.0.0-20201217121537-9736e1703824
	google.golang.org/grpc v1.35.0
	google.golang.org/protobuf v1.25.0
)

replace github.com/at-wat/ebml-go => github.com/goheadroom/ebml-go v0.14.2-0.20210224182821-cc65f65ab2a6
