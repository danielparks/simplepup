package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func sshAgent() ssh.AuthMethod {
	socketPath := os.Getenv("SSH_AUTH_SOCK")
	if socketPath == "" {
		log.Fatal("Fatal: no SSH agent running")
	}

	sshAgent, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("Fatal: connecting to SSH agent: %s", err)
	}

	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
}

// RemoteHTTP is a connection to a remote HTTP server over SSH
type RemoteHTTP struct {
	SSHClient  *ssh.Client
	SSHSession *ssh.Session
	HTTPClient *http.Client
}

// RemoteHTTPConnect makes a connection to a remote HTTP server
func RemoteHTTPConnect(username string, host string, port int) (*RemoteHTTP, error) {
	config := &ssh.ClientConfig{
		/// FIXME ssh.InsecureIgnoreHostKey
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            username,
		Auth: []ssh.AuthMethod{
			sshAgent(),
		},
	}

	sshClient, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return nil, err
	}

	// It seems that we have to have a session open for the port forward to work.
	sshSession, err := sshClient.NewSession()
	if err != nil {
		sshClient.Close()
		return nil, err
	}

	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				// sshClient.Dial returns a forwarded net.Conn to port on the remote.
				return sshClient.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			},
		},
	}

	remoteHTTP := &RemoteHTTP{
		SSHClient:  sshClient,
		SSHSession: sshSession,
		HTTPClient: &httpClient,
	}

	return remoteHTTP, nil
}

// Close the SSH session. Idempotent.
func (c *RemoteHTTP) Close() {
	c.HTTPClient = nil

	if c.SSHSession != nil {
		c.SSHSession.Close()
		c.SSHSession = nil
	}

	if c.SSHClient != nil {
		c.SSHClient.Close()
		c.SSHClient = nil
	}
}
