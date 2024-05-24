package reghive

import (
	"github.com/JoshuaDoes/crunchio"
)

type ABCDDevice struct {
	*crunchio.Buffer
}

func NewBCDDevice(b []byte) (*ABCDDevice, error) {
	if len(b) != 16 {
		return nil, ERROR_BCDDEVICE_HEADER_SIZE
	}

	dev := &ABCDDevice{crunchio.NewBuffer(b)}
	return dev, nil
}