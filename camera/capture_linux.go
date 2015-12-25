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
) (io.ReadCloser, func() error, error) {
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
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	wait := func() error {
		return cmd.Wait()
	}

	return rc, wait, nil
}
