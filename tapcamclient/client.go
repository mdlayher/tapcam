// Package tapcamclient provides a client for the tapcamd service.
package tapcamclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/sftp"
)

// A Client is a client for the tapcamd service.  It currently makes use of
// SFTP to upload images to tapcamd, but this may change in the future.
type Client struct {
	// Debug writer for SFTP client, ioutil.Discard by default
	debug io.Writer

	// SFTP client and function to close SSH session
	c     *sftp.Client
	cdone func() error
}

// New creates a new Client using the input host and zero or more optional
// Option functions.
func New(host string, options ...Option) (*Client, error) {
	c := &Client{
		debug: ioutil.Discard,
	}

	for _, o := range options {
		if err := o(c); err != nil {
			return nil, err
		}
	}

	sftpc, cdone, err := newClient(host, c.debug)
	if err != nil {
		return nil, err
	}

	c.c = sftpc
	c.cdone = cdone

	return c, nil
}

// An Option is a function which can apply configuration to a Client.
type Option func(c *Client) error

// newClient establishes SSH and SFTP connections to the specified host.
// TODO(mdlayher): use x/crypto/ssh directly instead of shelling out.
func newClient(host string, out io.Writer) (*sftp.Client, func() error, error) {
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

// Close closes the Client's internal network connections.
func (c *Client) Close() error {
	if err := c.c.Close(); err != nil {
		return err
	}

	return c.cdone()
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
