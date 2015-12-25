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
) (io.ReadCloser, func() error, error) {
	return nil, nil, errors.New("camera: capture not implemented")
}
