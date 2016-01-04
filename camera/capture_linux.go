// +build linux

package camera

import (
	"io"
	"os/exec"
)

func capture(
	device string,
	format Format,
	resolution *Resolution,
) (io.ReadCloser, error) {
	cmd := exec.Command(
		"streamer",
		"-c",
		device,
		"-f",
		string(format),
		"-o",
		"/dev/stdout",
		"-s",
		resolution.String(),
	)
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := func() error {
		return cmd.Wait()
	}

	return &readCloser{
		ReadCloser: rc,
		done:       done,
	}, nil
}
