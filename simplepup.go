package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
)

func main() {
	// Don't include metadata like the time in log messages.
	log.SetFlags(0)

	if len(os.Args) != 2 {
		log.Fatal("usage: simplepup query")
	}

	client, err := RemoteHTTPConnect("dp", "pdb.ops.puppetlabs.net", 8080)
	if err != nil {
		log.Fatalf("Error connecting to PuppetDB host: %s", err)
	}

	query := os.Args[1]
	queryURL := "http://localhost/pdb/query/v4?query=" + url.QueryEscape(query)
	resp, err := client.HTTPClient.Get(queryURL)
	if err != nil {
		log.Fatalf("Error connecting to PuppetDB: %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading PuppetDB response: %s", err)
	}

	fmt.Print(string(body))
}
