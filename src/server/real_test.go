package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type jsonError struct {
	Err string `json:"error"`
}

const url = "https://localhost:8080"

func get(path string) (*http.Response, error) {
	return client.Get(url + path)
}

func postJSONString(path string, jsonString string) (*http.Response, error) {
	return client.Post(url+path, "application/json",
		strings.NewReader(jsonString))
}

func postJSONBytes(path string, jsonBytes []byte) (*http.Response, error) {
	return client.Post(url+path, "application/json",
		bytes.NewReader(jsonBytes))
}

func getResponseString(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	var respBytes []byte
	respBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	} else {
		return string(respBytes), nil
	}
}

func getResponseBytes(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	var respBytes []byte
	respBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	} else {
		return respBytes, nil
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
	resp, err := postJSONString("/", "what")

	if err != nil {
		t.Error(err)
		return
	}

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Error(resp.StatusCode)
	}
}

var testAccounts = []accountCreateRequest{
	accountCreateRequest{Name: "John Doe", CPF: "220.321-11", Secret: "toto"},
	accountCreateRequest{Name: "Jane Doe", CPF: "221.321-11", Secret: "tata"},
	accountCreateRequest{Name: "Arseny", CPF: "222.321-13", Secret: "tete"},
}

func TestCreateAccounts(t *testing.T) {
	var respBytes []byte
	var err error
	var resp *http.Response
	var jsonBytes []byte
	var acc account
	var jsonErr jsonError

	for _, testAccount := range testAccounts {

		jsonBytes, err = json.Marshal(&testAccount)

		if err != nil {
			t.Error(err)
			return
		}

		resp, err = postJSONBytes("/accounts", jsonBytes)

		if err != nil {
			t.Error(err)
		} else if resp.StatusCode != http.StatusCreated {
			t.Error(resp.StatusCode)
			resp.Body.Close()
		} else {
			respBytes, err = getResponseBytes(resp)

			if err != nil {
				t.Error(err)
			} else {
				err = json.Unmarshal(respBytes, &acc)
				if err != nil {
					t.Error(err)
				} else if testAccount.CPF != acc.CPF {
					t.Error(acc.CPF)
				}
			}
		}

		// Re-insert the account with the same CPF
		resp, err = postJSONBytes("/accounts", jsonBytes)

		if err != nil {
			t.Error(err)
		} else if resp.StatusCode != http.StatusBadRequest {
			t.Error(resp.StatusCode)
			resp.Body.Close()
		} else {
			respBytes, err = getResponseBytes(resp)

			if err != nil {
				t.Error(err)
			} else {
				err = json.Unmarshal(respBytes, &jsonErr)

				if err != nil {
					t.Error(err)
				} else if jsonErr.Err != accExistsError.ErrMsg {
					t.Fail()
				}
			}
		}
	}
}

var badTestAccounts = []struct {
	accReq    accountCreateRequest
	publicErr *publicJSONError
}{
	{accountCreateRequest{Name: "John Doe", CPF: "220.321-111",
		Secret: "toto"},
		cpfInvalidError},
	{accountCreateRequest{
		Name:   "John Doeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		CPF:    "220.321-11",
		Secret: "toto"},
		nameTooLongError},
	{accountCreateRequest{
		Name:   "John Doe",
		CPF:    "220.321-111",
		Secret: "totoooooooooooooooooooooooooooooooooooooooooooooooooooooooo"},
		pwTooLongError},
	{accountCreateRequest{
		Name: "John Doe",
		CPF:  "220.321-111",
		Secret: `totoooooooooooooooooooooooooooooooooooooooooooooooooooooooo
		oooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo
		oooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo`},
		requestTooLongError},
}

func TestCreateBadAccounts(t *testing.T) {
	var respBytes []byte
	var err error
	var resp *http.Response
	var jsonBytes []byte
	var accReq accountCreateRequest
	var jsonErr jsonError

	for _, testAccountTuple := range badTestAccounts {

		accReq = testAccountTuple.accReq

		jsonBytes, err = json.Marshal(&accReq)

		if err != nil {
			t.Error(err)
			return
		}

		resp, err = postJSONBytes("/accounts", jsonBytes)

		if err != nil {
			t.Error(err)
		} else if resp.StatusCode != testAccountTuple.publicErr.Status {
			t.Error(resp.StatusCode)
			resp.Body.Close()
		} else {
			respBytes, err = getResponseBytes(resp)

			if err != nil {
				t.Error(err)
			} else {
				err = json.Unmarshal(respBytes, &jsonErr)

				if err != nil {
					t.Error(err)
				} else if jsonErr.Err != testAccountTuple.publicErr.ErrMsg {
					t.Error(jsonErr.Err)
				}
			}
		}
	}
}

// This is a big test that starts the server and talks to it with http.Client.
// It deletes stuff in the database to clear it first.
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
