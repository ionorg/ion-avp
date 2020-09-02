package elements

import (
	"os"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
)

const (
	// IDFileWriter .
	IDFileWriter = "FileWriter"
)

// FileWriterConfig .
type FileWriterConfig struct {
	ID   string
	Path string
}

// FileWriter instance
type FileWriter struct {
	id   string
	file *os.File
}

// NewFileWriter instance
func NewFileWriter(config FileWriterConfig) *FileWriter {
	w := &FileWriter{
		id: config.ID,
	}

	f, err := os.OpenFile(config.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		log.Errorf("error initializing filewriter: %s", err)
		return nil
	}

	w.file = f

	log.Infof("NewFileWriter with config: %+v", config)

	return w
}

// ID for FileWriter
func (w *FileWriter) ID() string {
	return IDFileWriter
}

func (w *FileWriter) Write(sample *avp.Sample) error {
	_, err := w.file.Write(sample.Payload.([]byte))
	return err
}

func (w *FileWriter) Read() <-chan *avp.Sample {
	return nil
}

// Attach attach a child element
func (w *FileWriter) Attach(e avp.Element) error {
	return ErrAttachNotSupported
}

// Close FileWriter
func (w *FileWriter) Close() {
	log.Infof("FileWriter.Close() %s", w.id)
}
