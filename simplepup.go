package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"strings"

	"github.com/kevinburke/ssh_config"
)

func getContentType(resp *http.Response) (string, string) {
	var contentTypeFull string
	if len(resp.Header["Content-Type"]) == 0 {
		contentTypeFull = "application/octet-stream"
	} else if len(resp.Header["Content-Type"]) == 1 {
		contentTypeFull = resp.Header["Content-Type"][0]
	} else {
		log.Fatal("Got more than one Content-Type header")
	}

	parts := strings.SplitN(contentTypeFull, ";", 2)
	return parts[0], parts[1]
}

func prettify(content []byte, contentType string) string {
	if contentType == "application/json" {
		var prettyBytes bytes.Buffer
		err := json.Indent(&prettyBytes, content, "", "  ")
		if err != nil {
			// Couldn't parse it as JSON, so just return the content unmodified.
			return string(content)
		}

		return prettyBytes.String()
	}

	return string(content)
}

func httpGet(client *RemoteHTTP, requestURL string) string {
	resp, err := client.HTTPClient.Get(requestURL)
	if err != nil {
		log.Fatalf("Error connecting to PuppetDB: %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading PuppetDB response: %s", err)
	}

	contentType, _ := getContentType(resp)
	stringBody := prettify(body, contentType)

	if resp.StatusCode == 400 {
		// Generally a PQL error.
		log.Fatal(stringBody)
	} else if resp.StatusCode != 200 {
		log.Fatalf("HTTP %s\n\n%s", resp.Status, stringBody)
	}

	return stringBody
}

func main() {
	// Don't include metadata like the time in log messages.
	log.SetFlags(0)

	if len(os.Args) != 2 {
		log.Fatal("usage: simplepup query")
	}

	nominalHostname := "pdb.ops.puppetlabs.net"
	hostname := ssh_config.Get(nominalHostname, "Hostname")
	if hostname == "" {
		hostname = nominalHostname
	}

	username := ssh_config.Get(nominalHostname, "User")
	if username == "" {
		currentUser, err := user.Current()
		if err != nil {
			log.Fatalf("Could not get current user: %s", err)
		}

		username = currentUser.Username
	}

	client, err := RemoteHTTPConnect(username, hostname, 8080)
	if err != nil {
		log.Fatalf("Error connecting to PuppetDB host: %s", err)
	}

	query := os.Args[1]
	queryURL := "http://localhost/pdb/query/v4?query=" + url.QueryEscape(query)
	fmt.Println(httpGet(client, queryURL))
}
