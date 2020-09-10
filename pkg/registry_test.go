package avp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type elementMock struct{}

func (e *elementMock) ID() string {
	return ""
}

func (e *elementMock) Write(*Sample) error {
	return nil
}

func (e *elementMock) Attach(Element) error {
	return nil
}

func (e *elementMock) Read() <-chan *Sample {
	return nil
}

func (e *elementMock) Close() {}

func testFunc(sid, pid, tid string, config []byte) Element {
	return &elementMock{}
}

func TestNewRegistry(t *testing.T) {

	registry := NewRegistry()
	assert.NotNil(t, registry)

	registry.AddElement("test", testFunc)
	expectedElement := registry.GetElement("test")

	assert.Equal(t, expectedElement("1", "2", "3", []byte{0x00}), testFunc("1", "2", "3", []byte{0x00}))
}
