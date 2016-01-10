// Package tapcamclient provides a client for the tapcamd service.
package tapcamclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// A Client is a client for the tapcamd service.  It currently makes use of
// SFTP to upload images to tapcamd, but this may change in the future.
type Client struct {
	// SFTP client and underlying SSH connection
	c    *sftp.Client
	conn *ssh.Client

	// SSH credentials
	sshUser string
	sshKey  ssh.Signer
}

// New creates a new Client using the input host and zero or more optional
// Option functions.
//
// At a minimum, the SSHUserKeyFile Option must be specified to allow
// authentication and image upload.
func New(host string, options ...Option) (*Client, error) {
	c := new(Client)
	for _, o := range options {
		if err := o(c); err != nil {
			return nil, err
		}
	}

	conn, err := ssh.Dial("tcp", host, &ssh.ClientConfig{
		User: c.sshUser,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(c.sshKey)},
	})
	if err != nil {
		return nil, err
	}

	sftpc, err := sftp.NewClient(conn)
	if err != nil {
		return nil, err
	}

	c.c = sftpc
	c.conn = conn

	return c, nil
}

// An Option is a function which can apply configuration to a Client.
type Option func(c *Client) error

// SSHUserKeyFile configures a Client to use an SSH username and private
// key file to authenticate against a tapcamd server.
func SSHUserKeyFile(user string, keyFile string) Option {
	return func(c *Client) error {
		keyBytes, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return err
		}

		private, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return err
		}

		c.sshUser = user
		c.sshKey = private
		return nil
	}
}

// Close closes the Client's internal network connections.
func (c *Client) Close() error {
	if err := c.c.Close(); err != nil {
		return err
	}

	return c.conn.Close()
}

// Upload uploads a file to the target location from an input io.Reader.
func (c *Client) Upload(target string, r io.Reader) error {
	target = filepath.Clean(target)

	// Make use of a temporary file to allow the entire upload to complete
	// before replacing the old file with this temporary one
	targetTemp := filepath.Clean(fmt.Sprintf("%s.%s",
		target,
		strconv.FormatInt(time.Now().UnixNano(), 10),
	))

	f, err := c.c.Create(targetTemp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, r); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	// Attempt to remove target file, but ignore error if it
	// doesn't exist
	if err := c.c.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}

	return c.c.Rename(targetTemp, target)
}
