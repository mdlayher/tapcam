package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/mdlayher/tapcam/camera"
	"github.com/pkg/sftp"
)

var (
	device = flag.String("d", camera.DefaultDevice, "webcam device location")
	format = flag.String("f", string(camera.FormatJPEG), "webcam image capture format")
	size   = flag.String("s", camera.Resolution1080p.String(), "webcam image size")
	delay  = flag.Int("delay", 0, "delay in seconds before taking a picture")
	host   = flag.String("host", "", "tapcamd host")
)

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func main() {
	flag.Parse()

	if *host == "" {
		log.Fatal("must specify SFTP host")
	}

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

	if d := *delay; d > 0 {
		log.Printf("taking picture in %d seconds", d)
		for i := 0; i < d; i++ {
			fmt.Print(". ")
			time.Sleep(1 * time.Second)
		}
		fmt.Println()
	}

	rc, done, err := cam.Capture()
	if err != nil {
		log.Fatal(err)
	}

	img, _, err := image.Decode(rc)
	if err != nil {
		log.Fatal(err)
	}

	if err := done(); err != nil {
		log.Fatal(err)
	}

	crop, ok := img.(subImager)
	if !ok {
		log.Fatal("cannot take sub image")
	}

	bounds := img.Bounds()
	bounds.Max = image.Point{X: 1920, Y: 100}

	simg := crop.SubImage(bounds)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, simg, nil); err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("out.jpg", buf.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}

	/*
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
	*/
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
