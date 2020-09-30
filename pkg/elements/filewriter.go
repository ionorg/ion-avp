package elements

import (
	"os"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
)

// FileWriter instance
type FileWriter struct {
	file *os.File
}

// NewFileWriter instance
func NewFileWriter(path string) *FileWriter {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		log.Errorf("error initializing filewriter: %s", err)
		return nil
	}

	return &FileWriter{
		file: f,
	}
}

func (w *FileWriter) Write(sample *avp.Sample) error {
	_, err := w.file.Write(sample.Payload.([]byte))
	return err
}

// Attach attach a child element
func (w *FileWriter) Attach(e avp.Element) {
	log.Warnf("FileWriter.Attach() not supported")
}

// Close FileWriter
func (w *FileWriter) Close() {}
