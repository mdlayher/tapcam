// +build !linux

package camera

import (
	"errors"
	"io"
)

func capture(
	device string,
	format Format,
	resolution *Resolution,
) (io.ReadCloser, error) {
	return nil, errors.New("camera: capture not implemented")
}
