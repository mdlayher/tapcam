// Package tapcamd provides a SSH and SFTP server for the tapcamd service.
package tapcamd

import (
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var (
	ErrNoConfig = errors.New("no SSH server configuration specified")
)

/*
Server:
  - public key to user authenticator
  - server host private key
  - SSH listen address
  -
*/

type Server struct {
	host   string
	dir    string
	config *ssh.ServerConfig
}

func New(host string, directory string, config *ssh.ServerConfig) (*Server, error) {
	if config == nil {
		return nil, ErrNoConfig
	}

	directory = filepath.Clean(directory)
	if _, err := os.Stat(directory); err != nil {
		return nil, err
	}

	s := &Server{
		host:   host,
		dir:    directory,
		config: config,
	}

	return s, nil
}

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
				s.dir,
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
