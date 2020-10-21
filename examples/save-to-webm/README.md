# save-to-webm
save-to-webm demonstrates how to process media with an `avp`. in this example, the media is relayed from `ion-sfu`, buffered and sequenced on the avp, and written to a webm file on disk.

## Instructions

### Start avp server
Run `go run examples/save-to-webm/server/main.go -c examples/save-to-webm/server/config.toml`. This will start an avp instance that will process media tracks.

### Start avp client
Run `go run examples/save-to-webm/client/main.go $SESSION_ID`. This will initiate a webrtc transport from avp to sfu for the given session. Tracks will start being relayed. When prompted, enter a track id to create a `WebmSaver` element which will start writing the track data to disk.

Congrats, you are now processing media with the ion-avp! Now start building something cool!
