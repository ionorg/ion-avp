<h1 align="center">
  <br>
  ion-avp
  <br>
</h1>
<h4 align="center">Go implementation of an Audio/Visual Processing Service</h4>
<p align="center">
  <a href="http://gophers.slack.com/messages/pion"><img src="https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen" alt="Slack Widget"></a>
  <a href="https://travis-ci.org/pion/ion-avp"><img src="https://travis-ci.org/pion/ion-avp.svg?branch=master" alt="Build Status"></a>
  <a href="https://pkg.go.dev/github.com/pion/ion-avp"><img src="https://godoc.org/github.com/pion/ion-avp?status.svg" alt="GoDoc"></a>
  <a href="https://codecov.io/gh/pion/ion-avp"><img src="https://codecov.io/gh/pion/ion-avp/branch/master/graph/badge.svg" alt="Coverage Status"></a>
  <a href="https://goreportcard.com/report/github.com/pion/ion-avp"><img src="https://goreportcard.com/badge/github.com/pion/ion-avp" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
</p>
<br>

ion-avp is an extensible audio/video processing service designed for use with ion-sfu.

## Getting Started

### Running the server

If you have a local golang environment already setup, simply do

```
go build cmd/main.go && ./main -c config.toml
```

If you prefer a containerized environment, you can use the included Docker image

```
docker build -t pionwebrtc/ion-avp .
docker run -p 50051:50051 -p 5000-5020:5000-5020/udp pionwebrtc/ion-avp:latest
```

### License

MIT License - see [LICENSE](LICENSE) for full text
