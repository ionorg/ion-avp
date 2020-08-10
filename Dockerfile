FROM golang:1.14.4-stretch

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/pion/ion-avp

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/pion/ion-avp && go mod download

COPY pkg/ $GOPATH/src/github.com/pion/ion-avp/pkg
COPY cmd/ $GOPATH/src/github.com/pion/ion-avp/cmd

WORKDIR $GOPATH/src/github.com/pion/ion-avp/cmd/server/grpc
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /avp .

FROM alpine:3.12.0

RUN apk --no-cache add ca-certificates
COPY --from=0 /avp /usr/local/bin/avp

COPY config.toml /configs/avp.toml

ENTRYPOINT ["/usr/local/bin/avp"]
CMD ["-c", "/configs/avp.toml"]
