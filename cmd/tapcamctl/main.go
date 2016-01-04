package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/mdlayher/tapcam/camera"
	"github.com/mdlayher/tapcam/tapcamclient"
)

var (
	device = flag.String("d", camera.DefaultDevice, "webcam device location")
	format = flag.String("f", string(camera.FormatJPEG), "webcam image capture format")
	size   = flag.String("s", camera.Resolution1080p.String(), "webcam image size")
	delay  = flag.Int("delay", 0, "delay in seconds before taking a picture")
	host   = flag.String("host", "", "tapcamd host")
)

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

	c, err := tapcamclient.New(*host)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Upload("/tmp/tapcam/latest.jpg", rc); err != nil {
		log.Fatal(err)
	}

	if err := rc.Close(); err != nil {
		log.Fatal(err)
	}

	if err := done(); err != nil {
		log.Fatal(err)
	}

	if err := c.Close(); err != nil {
		log.Fatal(err)
	}
}
