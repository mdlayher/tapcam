// Package tapcamd provides a SSH and SFTP server for the tapcamd service.
package tapcamd

import (
	"errors"
	"log"
	"net"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var (
	// ErrNoConfig is returned when a nil SSH server configuration is specified
	// when New is called.
	ErrNoConfig = errors.New("no SSH server configuration specified")
)

// A Server is a SSH and SFTP server for the tapcamd service.  It can be used
// to start a listener to accept SFTP connections, using authentication
// mechanisms specified when passing a *ssh.ServerConfig.
type Server struct {
	host   string
	config *ssh.ServerConfig
}

// New creates a new SSH and SFTP server which listens on the specified address
// and accepts a *ssh.ServerConfig to configure the underlying SSH server's
// properties and authentication mechanisms.
func New(host string, config *ssh.ServerConfig) (*Server, error) {
	if config == nil {
		return nil, ErrNoConfig
	}

	s := &Server{
		host:   host,
		config: config,
	}

	return s, nil
}

// ListenAndServe begins serving SSH and SFTP connections.
func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.host)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		_, chans, reqs, err := ssh.NewServerConn(conn, s.config)
		if err != nil {
			return err
		}

		// The incoming Request channel must be serviced.
		go ssh.DiscardRequests(reqs)

		// Service the incoming Channel channel.
		for c := range chans {
			if c.ChannelType() != "session" {
				c.Reject(ssh.UnknownChannelType, "unknown channel type")
				continue
			}
			channel, requests, err := c.Accept()
			if err != nil {
				log.Println("failed to accept connection:", err)
				continue
			}

			// Sessions have out-of-band requests such as "shell",
			// "pty-req" and "env".  Here we handle only the
			// "subsystem" request.
			go func(in <-chan *ssh.Request) {
				for req := range in {
					if req.Type != "subsystem" || string(req.Payload[4:]) != "sftp" {
						_ = req.Reply(false, nil)
						return
					}

					_ = req.Reply(true, nil)
				}
			}(requests)

			server, err := sftp.NewServer(
				channel,
				channel,
				// TODO(mdlayher): this option does nothing, and will be
				// removed in a PR to github.com/pkg/sftp
				"/",
			)
			if err != nil {
				return err
			}
			if err := server.Serve(); err != nil {
				return err
			}
		}
	}

	return nil
}
