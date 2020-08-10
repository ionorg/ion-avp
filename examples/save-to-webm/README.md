# process-on-avp
process-on-avp demonstrates how to process media with an avp. in this example, the media is pubished to ion-sfu from a file on disk.

## Instructions
### Create IVF named `output.ivf` that contains a VP8 track and/or `output.ogg` that contains a Opus track
```
ffmpeg -i $INPUT_FILE -g 30 output.ivf
ffmpeg -i $INPUT_FILE -c:a libopus -page_duration 20000 -vn output.ogg
```

### Download process-on-avp
```
go get github.com/pion/ion-sfu/examples/process-on-avp
```

### Run process-on-avp
The `output.ivf` you created should be in the same directory as `process-on-avp`.

Run `process-on-avp $yourroom < my_file`

Congrats, you are now publishing video to the ion-sfu! Now start building something cool!
