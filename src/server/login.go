package server

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type loginRequest struct {
	CPF    string `json:"cpf"`
	Secret string `json:"secret"`
}

// We can use the login time to timeout the requests.
type userEntry struct {
	id        int
	loginTime time.Time
}

type userMap struct {
	entries map[string]userEntry
	mu      sync.Mutex
}

// Our in-memory map of users. Remember to use the mutex to access, as many
// parallel requests can use it.
var users userMap = userMap{entries: make(map[string]userEntry, 64)}

// Generate a token. The token are the hex-encoded first 8 bytes of a sha256
// of a randomly-generated number. I'm not actually sure if this is secure.
func generateToken() (string, error) {
	val, err := rand.Int(rand.Reader, big.NewInt(2147483647))

	if err != nil {
		fmt.Println(err)
		return "", err
	}

	token := sha256.Sum256(
		[]byte(
			fmt.Sprintf("%x", val.Int64())))

	return fmt.Sprintf("%x", token[0:8]), nil
}

// Handler for POST at /login. Retrieves the account using the CPF in the
// request, checks if the request password matches the account secret,
// and if so inserts a new logged-in user to our in-memory map of logged-in
// users, and returns the token.
func login(rw http.ResponseWriter, req *http.Request) {

	if req.URL.Path != "/login" {
		respondWithError(rw, invalidURLError)
		return
	}

	if req.Method != http.MethodPost {
		respondWithError(rw, invalidMethodError)
		return
	}

	var loginReq loginRequest
	var data, err = readFromReq(req, 128)

	if err != nil {
		respondWithError(rw, err)
		return
	}

	err = json.Unmarshal(data, &loginReq)

	if err != nil {
		respondWithError(rw, cantParseJSONError)
		return
	}

	if !cpfRegex.MatchString(loginReq.CPF) {
		respondWithError(rw, cpfInvalidError)
		return
	}

	var acc account

	row := db.QueryRow(
		"select id, secret from accounts where cpf = $1", loginReq.CPF)

	err = row.Scan(&acc.ID, &acc.secret)

	if err == sql.ErrNoRows {
		respondWithError(rw, noAccountError)
		return
	} else if err != nil {
		respondWithError(rw, err)
		return
	}

	hashedSecret := fmt.Sprintf("%x", sha256.Sum256([]byte(loginReq.Secret)))

	if hashedSecret != acc.secret {
		respondWithError(rw, wrongPasswordError)
		return
	}

	var token string
	token, err = generateToken()

	if err != nil {
		respondWithError(rw, err)
		return
	}

	users.mu.Lock()
	_, present := users.entries[token]
	if !present {
		users.entries[token] = userEntry{acc.ID, time.Now()}
	}
	users.mu.Unlock()

	// We randomly re-generated the same token, return with http conflict,
	// so that the user knows to try again. If we added the user, a previously
	// logged in user would have access to this user!
	if present {
		respondWithError(rw, tryAgainError)
		return
	}

	rw.WriteHeader(http.StatusCreated)
	setJSONEncoding(rw)

	_, err = rw.Write([]byte(fmt.Sprintf("{\"token\":\"%s\"}\n", token)))

	if err != nil {
		logger.Printf("Could not write response: %v", err)
	}
}
