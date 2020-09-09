package avp

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

func TestNewRegistry(t *testing.T) {

	registry := NewRegistry()
	assert.NotNil(t, registry)

	testFunc := func(sid, pid, tid string) Element {
		return &elementMock{}
	}

	registry.AddElement("test", testFunc)
	expectedElement := registry.GetElement("test")

	assert.Equal(t, expectedElement("1", "2", "3"), testFunc("1", "2", "3"))
}
