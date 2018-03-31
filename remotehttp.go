package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
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

func filterNonExistantPaths(allPaths []string) []string {
	goodPaths := []string{}
	for _, path := range allPaths {
		_, err := os.Stat(path)
		if err == nil {
			goodPaths = append(goodPaths, path)
		}
	}
	return goodPaths
}

func standardKnownHosts() (ssh.HostKeyCallback, error) {
	knownHostsPaths := []string{
		os.Getenv("HOME") + "/.ssh/known_hosts",
		"/etc/ssh/ssh_known_hosts",
	}

	knownHostsPaths = filterNonExistantPaths(knownHostsPaths)

	if len(knownHostsPaths) == 0 {
		knownHostsPaths = append(knownHostsPaths, "/dev/null")
	}

	return knownhosts.New(knownHostsPaths...)
}

// RemoteHTTP is a connection to a remote HTTP server over SSH
type RemoteHTTP struct {
	SSHClient  *ssh.Client
	SSHSession *ssh.Session
	HTTPClient *http.Client
}

// RemoteHTTPConnect makes a connection to a remote HTTP server
func RemoteHTTPConnect(username string, host string, httpPort int) (*RemoteHTTP, error) {
	knownHostsCallback, err := standardKnownHosts()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		HostKeyCallback: knownHostsCallback,
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
		Timeout: time.Second * 60,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				// sshClient.Dial returns a tunneled net.Conn to httpPort on the remote.
				return sshClient.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", httpPort))
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
