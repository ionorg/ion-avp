package elements

import (
	"os"

	avp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
)

// FileWriter instance
type FileWriter struct {
	Leaf
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
