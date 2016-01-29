package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"time"

	"github.com/mdlayher/tapcam/camera"
	"github.com/mdlayher/tapcam/tapcamclient"

	"github.com/disintegration/imaging"
)

const (
	envHost    = "TAPCAMCTL_HOST"
	envUser    = "TAPCAMCTL_USER"
	envKeyFile = "TAPCAMCTL_KEY"
	envTarget  = "TAPCAMCTL_TARGET"
	envInvert  = "TAPCAMCTL_INVERT"
)

var (
	device = flag.String("d", camera.DefaultDevice, "webcam device location")
	format = flag.String("f", string(camera.FormatJPEG), "webcam image capture format")
	size   = flag.String("s", camera.Resolution1080p.String(), "webcam image size")
	delay  = flag.Int("delay", 0, "delay in seconds before taking a picture")

	host    = flag.String("host", "", "tapcamd host")
	user    = flag.String("user", "", "tapcamd SSH user")
	keyFile = flag.String("key", "", "tapcamd SSH private key file")
	target  = flag.String("target", "", "tapcamd image upload target location")
	invert  = flag.Bool("i", false, "invert image after capture")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	flag.Parse()

	config, err := checkFlags(
		*host,
		*user,
		*keyFile,
		*target,
		*invert,
	)
	if err != nil {
		log.Fatal(err)
	}

	tcc, err := tapcamclient.New(
		config.host,
		tapcamclient.SSHUserKeyFile(config.user, config.keyFile),
	)
	if err != nil {
		log.Fatal(err)
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
		log.Printf("capturing image in %d seconds", d)
		for i := 0; i < d; i++ {
			fmt.Print(". ")
			time.Sleep(1 * time.Second)
		}
		fmt.Println()
	}

	log.Println("capturing image with camera")

	rc, err := cam.Capture()
	if err != nil {
		log.Fatal(err)
	}

	if !config.invert {
		log.Println("uploading image to server")

		if err := tcc.Upload(config.target, rc); err != nil {
			log.Fatal(err)
		}

		if err := tcc.Close(); err != nil {
			log.Fatal(err)
		}

		log.Println("done!")
		return
	}

	log.Println("inverting image before upload")

	img, _, err := image.Decode(rc)
	if err != nil {
		log.Fatal(err)
	}

	if err := rc.Close(); err != nil {
		log.Fatal(err)
	}

	img = imaging.FlipV(img)
	img = imaging.FlipH(img)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		log.Fatal(err)
	}

	log.Println("uploading image to server")

	if err := tcc.Upload(config.target, &buf); err != nil {
		log.Fatal(err)
	}

	if err := tcc.Close(); err != nil {
		log.Fatal(err)
	}

	log.Println("done!")
}

type config struct {
	host    string
	user    string
	keyFile string
	target  string
	invert  bool
}

func checkFlags(
	host string,
	user string,
	keyFile string,
	target string,
	invert bool,
) (*config, error) {
	c := new(config)

	if host == "" {
		host = os.Getenv(envHost)
	}
	if host == "" {
		return nil, fmt.Errorf("must specify SFTP host using -host flag or $%s", envHost)
	}
	c.host = host

	if user == "" {
		user = os.Getenv(envUser)
	}
	if user == "" {
		return nil, fmt.Errorf("must specify SFTP user using -user flag or $%s", envUser)
	}
	c.user = user

	if keyFile == "" {
		keyFile = os.Getenv(envKeyFile)
	}
	if keyFile == "" {
		return nil, fmt.Errorf("must specify SFTP private key file using -key flag or $%s", envKeyFile)
	}
	c.keyFile = keyFile

	if target == "" {
		target = os.Getenv(envTarget)
	}
	if target == "" {
		return nil, fmt.Errorf("must specify SFTP file target using -target flag or $%s", envTarget)
	}
	c.target = target

	if !invert {
		invert = os.Getenv(envInvert) == "true"
	}
	c.invert = invert

	return c, nil
}
