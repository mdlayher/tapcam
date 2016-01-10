// Package camera provides webcam image capture functionality using
// the Linux streamer utility.  Image capture is not implemented on
// other platforms, and Camera.Capture will return an error if capture
// is not implemented.
package camera

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	// DefaultDevice is the default camera device used when no camera
	// device is specified.
	DefaultDevice = "/dev/video0"
)

var (
	// ErrInvalidFormat is returned when an invalid Format is selected.
	ErrInvalidFormat = errors.New("camera: invalid format")

	// ErrInvalidResolution is returned when a resolution string is not
	// in the expected "XXXxYYY" format.
	ErrInvalidResolution = errors.New("camera: invalid resolution")
)

// A Format is an image capture format.
type Format string

const (
	// FormatJPEG produces JPEG images.
	FormatJPEG Format = "jpeg"
)

// A Resolution is a X and Y specifying the width and height in pixels
// of a captured image.
type Resolution struct {
	X int
	Y int
}

// NewResolution creates a Resolution from a string in the format "XXXxYYY".
func NewResolution(res string) (*Resolution, error) {
	ss := strings.Split(res, "x")
	if len(ss) != 2 {
		return nil, ErrInvalidResolution
	}

	x, err := strconv.Atoi(ss[0])
	if err != nil {
		return nil, err
	}

	y, err := strconv.Atoi(ss[1])
	if err != nil {
		return nil, err
	}

	return &Resolution{
		X: x,
		Y: y,
	}, nil
}

// String returns the string representation of a Resolution.
func (r *Resolution) String() string {
	return fmt.Sprintf("%dx%d", r.X, r.Y)
}

var (
	// Resolution720p captures images using 720p resolution.
	Resolution720p = &Resolution{X: 1280, Y: 720}

	// Resolution1080p captures images using 1080p resolution.
	Resolution1080p = &Resolution{X: 1920, Y: 1080}
)

// A Camera is a camera device which can be used to capture images.
type Camera struct {
	device     string
	format     Format
	resolution *Resolution
}

// New creates a new Camera using zero or more Option functions.
func New(options ...Option) (*Camera, error) {
	// Apply any input options
	c := new(Camera)
	for _, o := range options {
		if err := o(c); err != nil {
			return nil, err
		}
	}

	// Specify default camera device if none set
	if c.device == "" {
		if err := SetDevice(DefaultDevice)(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Capture captures an image stream from a camera device.
// stream is an io.ReadCloser which contains the image stream.
func (c *Camera) Capture() (stream io.ReadCloser, err error) {
	return capture(
		c.device,
		c.format,
		c.resolution,
	)
}

// An Option is a function which can apply a configuration to a Camera.
type Option func(c *Camera) error

// SetDevice sets a video camera device for use with a Camera.
// It verifies that the device exists before returning.
func SetDevice(device string) Option {
	return func(c *Camera) error {
		if _, err := os.Stat(device); err != nil {
			return err
		}

		c.device = device
		return nil
	}
}

// SetFormat sets an image capture format for use with a Camera.
// It verifies that the Format is one specified in the Format constants.
func SetFormat(format Format) Option {
	return func(c *Camera) error {
		if format != FormatJPEG {
			return ErrInvalidFormat
		}

		c.format = format
		return nil
	}
}

// SetResolution sets an image capture resolution for use with a Camera.
func SetResolution(resolution *Resolution) Option {
	return func(c *Camera) error {
		c.resolution = resolution
		return nil
	}
}

var _ io.ReadCloser = &readCloser{}

// readCloser is a special io.ReadCloser which wraps the done function
// created in capture, calling it after closing the wrapped io.ReadCloser.
type readCloser struct {
	io.ReadCloser
	done func() error
}

func (r *readCloser) Close() error {
	if err := r.ReadCloser.Close(); err != nil {
		return err
	}

	return r.done()
}
