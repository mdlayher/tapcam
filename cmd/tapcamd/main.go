package main

import (
	"crypto/subtle"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/mdlayher/tapcam/tapcamd"

	"golang.org/x/crypto/ssh"
)

var (
	httpHost = flag.String("http", ":8080", "host address for HTTP server")
	sftpHost = flag.String("sftp", ":2222", "host address for SFTP server")
	imageDir = flag.String("d", "/tmp/tapcam", "directory for image storage")

	sftpHostKey   = flag.String("sftp-hostkey", "", "host private key file location for SFTP server")
	sftpUser      = flag.String("sftp-user", "", "username for SFTP server authentication")
	sftpPublicKey = flag.String("sftp-pubkey", "", "user public key file location for SFTP server authentication")
)

func main() {
	flag.Parse()

	if err := os.MkdirAll(*imageDir, 0755); err != nil {
		log.Fatal(err)
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir(*imageDir)))

		log.Printf("starting HTTP server at %q", *httpHost)
		if err := http.ListenAndServe(*httpHost, mux); err != nil {
			log.Fatal(err)
		}
	}()

	pka, err := publicKeyAuth(*sftpUser, *sftpPublicKey)
	if err != nil {
		log.Fatal(err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: pka,
	}
	if err := configureHostKey(*sftpHostKey, config); err != nil {
		log.Fatal(err)
	}

	s, err := tapcamd.New(*sftpHost, config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting SFTP server at %q", *sftpHost)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

type publicKeyAuthFunc func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

func publicKeyAuth(user string, file string) (publicKeyAuthFunc, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(buf)
	if err != nil {
		return nil, err
	}

	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		userOK := subtle.ConstantTimeCompare(
			[]byte(conn.User()),
			[]byte(user),
		) == 1
		keyOK := subtle.ConstantTimeCompare(
			key.Marshal(),
			pubKey.Marshal(),
		) == 1

		if userOK && keyOK {
			log.Printf("accepted authentication for user %q at %q",
				conn.User(), conn.RemoteAddr().String())
			return nil, nil
		}

		log.Printf("rejected authentication for user %q at %q",
			conn.User(), conn.RemoteAddr().String())
		return nil, fmt.Errorf("pubkey for %q not acceptable", conn.User())
	}, nil
}

func configureHostKey(file string, config *ssh.ServerConfig) error {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	hostKey, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return err
	}

	config.AddHostKey(hostKey)
	return nil
}
