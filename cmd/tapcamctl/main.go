package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mdlayher/tapcam/camera"
	"github.com/mdlayher/tapcam/tapcamclient"
	"golang.org/x/crypto/ssh"
)

const (
	envHost    = "TAPCAMCTL_HOST"
	envUser    = "TAPCAMCTL_USER"
	envKeyFile = "TAPCAMCTL_KEY"
)

var (
	device = flag.String("d", camera.DefaultDevice, "webcam device location")
	format = flag.String("f", string(camera.FormatJPEG), "webcam image capture format")
	size   = flag.String("s", camera.Resolution1080p.String(), "webcam image size")
	delay  = flag.Int("delay", 0, "delay in seconds before taking a picture")

	host    = flag.String("host", "", "tapcamd host")
	user    = flag.String("user", "", "tapcamd SSH user")
	keyFile = flag.String("key", "", "tapcamd SSH private key file")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	flag.Parse()

	host, user, keyFile, err := checkFlags(*host, *user, *keyFile)
	if err != nil {
		log.Fatal(err)
	}

	keyBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Fatal(err)
	}

	private, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		log.Fatal(err)
	}

	tcc, err := tapcamclient.New(host, &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(private)},
	})
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
		log.Printf("taking picture in %d seconds", d)
		for i := 0; i < d; i++ {
			fmt.Print(". ")
			time.Sleep(1 * time.Second)
		}
		fmt.Println()
	}

	rc, err := cam.Capture()
	if err != nil {
		log.Fatal(err)
	}

	if err := tcc.Upload("/tmp/tapcam/latest.jpg", rc); err != nil {
		log.Fatal(err)
	}

	if err := rc.Close(); err != nil {
		log.Fatal(err)
	}

	if err := tcc.Close(); err != nil {
		log.Fatal(err)
	}
}

func checkFlags(host string, user string, keyFile string) (string, string, string, error) {
	if host == "" {
		host = os.Getenv(envHost)
	}
	if host == "" {
		return "", "", "", fmt.Errorf("must specify SFTP host using -host flag or $%s", envHost)
	}

	if user == "" {
		user = os.Getenv(envUser)
	}
	if user == "" {
		return "", "", "", fmt.Errorf("must specify SFTP user using -user flag or $%s", envUser)
	}

	if keyFile == "" {
		keyFile = os.Getenv(envKeyFile)
	}
	if keyFile == "" {
		return "", "", "", fmt.Errorf("must specify SFTP private key file using -key flag or $%s", envKeyFile)
	}

	return host, user, keyFile, nil
}
