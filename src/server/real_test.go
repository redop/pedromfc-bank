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

type tokenResponse struct {
	Token string `json:"token"`
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

func TestCreateAccounts(t *testing.T) {
	var respBytes []byte
	var err error
	var resp *http.Response
	var jsonBytes []byte
	var acc account
	var jsonErr jsonError

	var testAccounts = []accountCreateRequest{
		accountCreateRequest{
			Name: "John Doe", CPF: "220.321-11", Secret: "toto"},
		accountCreateRequest{
			Name: "Jane Doe", CPF: "221.321-11", Secret: "tata"},
		accountCreateRequest{
			Name: "Arseny", CPF: "222.321-13", Secret: "tete"},
	}

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
				} else if jsonErr.Err != accExistsError.errMsg {
					t.Fail()
				}
			}
		}
	}
}

func TestCreateBadAccounts(t *testing.T) {
	var respBytes []byte
	var err error
	var resp *http.Response
	var jsonBytes []byte
	var accReq accountCreateRequest
	var jsonErr jsonError

	var badTestAccounts = []struct {
		accReq    accountCreateRequest
		publicErr *publicJSONError
	}{
		{accountCreateRequest{Name: "John Doe", CPF: "220.321-111",
			Secret: "toto"},
			cpfInvalidError},
		{accountCreateRequest{
			Name:   "John Doeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeooo",
			CPF:    "220.321-11",
			Secret: "toto"},
			nameTooLongError},
		{accountCreateRequest{
			Name:   "John Doe",
			CPF:    "220.321-111",
			Secret: "totoooooooooooooooooooooooooooooooooooooooooooooooooooo"},
			pwTooLongError},
		{accountCreateRequest{
			Name: "John Doe",
			CPF:  "220.321-111",
			Secret: `totooooooooooooooooooooooooooooooooooooooooooooooooooooooo
			ooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo
			ooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo
			ooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo
			oooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo`},
			requestTooLongError},
	}

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
		} else if resp.StatusCode != testAccountTuple.publicErr.status {
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
				} else if jsonErr.Err != testAccountTuple.publicErr.errMsg {
					t.Error(jsonErr.Err)
				}
			}
		}
	}
}

func TestCreateTransfers(t *testing.T) {
	var respBytes []byte
	var err error
	var resp *http.Response
	var jsonBytes []byte
	var accs [2]account //src, dst

	// Create two accounts for transfering
	var testAccounts = []accountCreateRequest{
		accountCreateRequest{
			Name: "John Doe", CPF: "320.321-11", Secret: "toto"},
		accountCreateRequest{
			Name: "Jane Doe", CPF: "321.321-11", Secret: "tata"},
	}

	for i, testAccount := range testAccounts {

		jsonBytes, err = json.Marshal(&testAccount)

		if err != nil {
			t.Log(err)
			t.FailNow()
		}

		resp, err = postJSONBytes("/accounts", jsonBytes)

		if err != nil {
			t.Log(err)
			t.FailNow()
		}

		if resp.StatusCode != http.StatusCreated {
			resp.Body.Close()
			t.Log(resp.StatusCode)
			t.FailNow()
		}

		respBytes, err = getResponseBytes(resp)

		if err != nil {
			t.Log(err)
			t.FailNow()
		}

		err = json.Unmarshal(respBytes, &accs[i])

		if err != nil {
			t.Log(err)
			t.FailNow()
		}
	}

	// Login as the first user
	loginReq := loginRequest{
		CPF:    testAccounts[0].CPF,
		Secret: testAccounts[0].Secret}

	jsonBytes, err = json.Marshal(&loginReq)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	resp, err = postJSONBytes("/login", jsonBytes)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Log(resp.StatusCode)
		t.FailNow()
	}

	respBytes, err = getResponseBytes(resp)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	var tokJSON tokenResponse
	err = json.Unmarshal(respBytes, &tokJSON)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	// Transfer to second user

	var transfReq = transferRequest{DestinationID: accs[1].ID,
		Amount: 33452}

	jsonBytes, err = json.Marshal(&transfReq)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	var req *http.Request
	req, err = http.NewRequest(http.MethodPost, url+"/transfers",
		bytes.NewReader(jsonBytes))

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	req.Header.Set("Authorization", tokJSON.Token)

	resp, err = client.Do(req)

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	if resp.StatusCode != http.StatusCreated {
		t.Log(resp.StatusCode)
		resp.Body.Close()
		t.FailNow()
	}

	respBytes, err = getResponseBytes(resp)

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	var transf transfer
	err = json.Unmarshal(respBytes, &transf)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if transf.Amount != transfReq.Amount {
		t.Log(transf.Amount)
		t.FailNow()
	}

	// Get the list of accounts to check their balance
	resp, err = get("/accounts")

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	respBytes, err = getResponseBytes(resp)

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	var allAccs []account
	err = json.Unmarshal(respBytes, &allAccs)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	for _, acc := range allAccs {
		if acc.ID == accs[0].ID {
			if acc.Balance != accs[0].Balance-transf.Amount {
				t.Log(acc.Balance)
				t.Fail()
			}
		} else if acc.ID == accs[1].ID {
			if acc.Balance != accs[0].Balance+transf.Amount {
				t.Log(acc.Balance)
				t.Fail()
			}
		}
	}

	// Get the account balance to check it
	resp, err = get(fmt.Sprintf("/accounts/%d/balance", accs[1].ID))

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	respBytes, err = getResponseBytes(resp)

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	var balanceResp accountBalanceResponse
	err = json.Unmarshal(respBytes, &balanceResp)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if balanceResp.Balance != accs[0].Balance+transf.Amount {
		t.Log(balanceResp.Balance)
		t.Fail()
	}

	// Get the list of transfers
	req, err = http.NewRequest(http.MethodGet, url+"/transfers",
		bytes.NewReader(jsonBytes))

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	req.Header.Set("Authorization", tokJSON.Token)

	resp, err = client.Do(req)

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	respBytes, err = getResponseBytes(resp)

	if err != nil {
		t.Log(err)
		resp.Body.Close()
		t.FailNow()
	}

	var transfers []transfer
	err = json.Unmarshal(respBytes, &transfers)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if len(transfers) != 1 {
		t.Log(err)
		t.FailNow()
	}

	if transfers[0].OriginID != accs[0].ID {
		t.Log(transfers[0].OriginID)
		t.Fail()
	}

	if transfers[0].DestinationID != accs[1].ID {
		t.Log(transfers[0].DestinationID)
		t.Fail()
	}

	if transfers[0].Amount != transfReq.Amount {
		t.Log(transfers[0].Amount)
		t.Fail()
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

	// Clean up the transfers before we test.
	_, err = DB.Exec("delete from transfers")

	if err != nil {
		fmt.Printf("Could not delete accounts: %v\n", err)
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
