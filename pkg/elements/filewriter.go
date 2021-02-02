package elements

import (
	"bufio"
	"io"
	"os"

	avp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
)

// FileWriter instance
type FileWriter struct {
	Leaf
	wr   io.Writer
	path string
}

// NewFileWriter instance
// bufSize is the buffer size in bytes. Pass <=0 to disable buffering.
func NewFileWriter(path string, bufSize int) *FileWriter {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		log.Errorf("error initializing filewriter: %s", err)
		return nil
	}

	fw := &FileWriter{
		path: path,
	}
	if bufSize > 0 {
		fw.wr = bufio.NewWriterSize(f, bufSize)
	} else {
		fw.wr = f
	}
	log.Infof("FileWriter opened %s", path)
	return fw
}

func (w *FileWriter) Write(sample *avp.Sample) error {
	_, err := w.wr.Write(sample.Payload.([]byte))
	return err
}

func (w *FileWriter) Close() {
	if c, ok := w.wr.(*bufio.Writer); ok {
		c.Flush()
	}
	if c, ok := w.wr.(io.Closer); ok {
		c.Close()
	}
	log.Infof("FileWriter closed %s", w.path)
}
