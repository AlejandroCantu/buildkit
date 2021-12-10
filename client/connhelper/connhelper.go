// Package connhelper provides helpers for connecting to a remote daemon host
// with custom logic.
package connhelper

import (
	"context"
	"net"
	"net/url"

	"github.com/docker/cli/cli/connhelper/commandconn"
	"github.com/docker/cli/cli/connhelper/ssh"
	"github.com/pkg/errors"
)

var helpers = map[string]func(*url.URL) (*ConnectionHelper, error){}

// ConnectionHelper allows to connect to a remote host with custom stream provider binary.
type ConnectionHelper struct {
	// ContextDialer can be passed to grpc.WithContextDialer
	ContextDialer func(ctx context.Context, addr string) (net.Conn, error)
}

// GetConnectionHelperWithSSHOpts returns Docker-specific connection helper for
// the given URL, and accepts additional options for ssh connections. It returns
// nil without error when no helper is registered for the scheme.
//
// Requires Docker 18.09 or later on the remote host.
func GetConnectionHelperWithSSHOpts(daemonURL string, sshFlags []string) (*ConnectionHelper, error) {
	return getConnectionHelper(daemonURL, sshFlags)
}

// GetConnectionHelper returns BuildKit-specific connection helper for the given URL.
// GetConnectionHelper returns nil without error when no helper is registered for the scheme.
func GetConnectionHelper(daemonURL string) (*ConnectionHelper, error) {
	return getConnectionHelper(daemonURL, nil)
}

func getConnectionHelper(daemonURL string, sshFlags []string) (*ConnectionHelper, error) {
	u, err := url.Parse(daemonURL)
	if err != nil {
		return nil, err
	}

	// Written for https://github.com/moby/buildkit/issues/2032
	if u.Scheme == "ssh" {
		sp, err := ssh.SpecFromURL(daemonURL)
		if err != nil {
			return nil, errors.Wrap(err, "ssh host connection is not valid")
		}
		return &ConnectionHelper{
			Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return commandconn.New(ctx, "ssh", append(sshFlags, sp.Args("docker", "system", "dial-stdio")...)...)
			},
			Host: "http://docker.example.com",
		}, nil
	}

	fn, ok := helpers[u.Scheme]
	if !ok {
		return nil, nil
	}

	return fn(u)
}



// Register registers new connectionhelper for scheme
func Register(scheme string, fn func(*url.URL) (*ConnectionHelper, error)) {
	helpers[scheme] = fn
}
