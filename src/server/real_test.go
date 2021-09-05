package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const url = "https://localhost:8080"

func get(path string) (*http.Response, error) {
	return client.Get(url + path)
}

func postJSON(path string, jsonString string) (*http.Response, error) {
	return client.Post(url+path, "application/json",
		strings.NewReader(jsonString))
}

func getResponseString(resp *http.Response) (string, error) {
	var respBytes []byte
	respBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	} else {
		return string(respBytes), nil
	}
}

var client http.Client

func TestGetWelcome(t *testing.T) {
	var respString string

	resp, err := get("/")

	if err != nil {
		t.Error(err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		t.Error(resp.StatusCode)
	}

	respString, err = getResponseString(resp)

	if err != nil {
		t.Error(err)
		return
	}

	if respString != welcomeSring {
		t.Error("got unexpected welcome string")
		return
	}
}

func TestBadURL(t *testing.T) {
	resp, err := get("/zzz")

	if err != nil {
		t.Error(err)
		return
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Error(resp.StatusCode)
	}
}

func TestWelcomeBadMethod(t *testing.T) {
	resp, err := postJSON("/", "what")

	if err != nil {
		t.Error(err)
		return
	}

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Error(resp.StatusCode)
	}
}

func TestMain(m *testing.M) {

	// We don't care about authentication for these tests.
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: tlsConfig}
	client = http.Client{Transport: tr}

	var err = OpenDBPool()
	defer DB.Close()

	if err != nil {
		fmt.Println("Could not open DB.")
		os.Exit(1)
	}

	// Clean up the accounts before we test.
	_, err = DB.Exec("delete from accounts")

	if err != nil {
		fmt.Printf("Could not delete accounts: %v\n", err)
		os.Exit(1)
	}

	go Run("../../certs")

	var resp *http.Response

	// Wait for the server to be responsive.
	for {
		time.Sleep(50 * time.Millisecond) // give it some time to start
		resp, err = get("/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
	}

	code := m.Run()

	Stop()
	<-ServerFinished

	os.Exit(code)
}
