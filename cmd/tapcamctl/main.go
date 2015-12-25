package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/mdlayher/tapcam/camera"
	"github.com/pkg/sftp"
)

var (
	device = flag.String("d", camera.DefaultDevice, "webcam device location")
	format = flag.String("f", string(camera.FormatJPEG), "webcam image capture format")
	size   = flag.String("s", camera.Resolution720p.String(), "webcam image size")
	host   = flag.String("host", "", "tapcamd host")
)

func main() {
	flag.Parse()

	resolution, err := camera.NewResolution(*size)
	if err != nil {
		log.Fatal(err)
	}

	cam, err := camera.New(
		camera.SetDevice(*device),
		camera.SetFormat(camera.Format(*format)),
		camera.SetResolution(resolution),
	)
	if err != nil {
		log.Fatal(err)
	}

	rc, done, err := cam.Capture()
	if err != nil {
		log.Fatal(err)
	}

	c, cdone, err := sftpClient(*host, ioutil.Discard)
	if err != nil {
		log.Fatal(err)
	}

	const (
		dir = "/tmp/tapcam"

		tmpName  = "latest.tmp.jpg"
		permName = "latest.jpg"
	)

	tmpFullName := filepath.Join(dir, tmpName)
	permFullName := filepath.Join(dir, permName)

	f, err := c.Create(tmpFullName)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := io.Copy(f, rc); err != nil {
		log.Fatal(err)
	}

	if err := done(); err != nil {
		log.Fatal(err)
	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	if err := c.Remove(permFullName); err != nil {
		log.Fatal(err)
	}

	if err := c.Rename(tmpFullName, permFullName); err != nil {
		log.Fatal(err)
	}

	if err := c.Close(); err != nil {
		log.Fatal(err)
	}

	if err := cdone(); err != nil {
		log.Fatal(err)
	}
}

func sftpClient(host string, out io.Writer) (*sftp.Client, func() error, error) {
	// Connect to a remote host and request the sftp subsystem via the 'ssh'
	// command.  This assumes that passwordless login is correctly configured.
	cmd := exec.Command("ssh", host, "-s", "sftp")
	cmd.Stderr = out

	wr, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	rd, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	c, err := sftp.NewClientPipe(rd, wr)
	return c, func() error {
		return cmd.Wait()
	}, err
}
