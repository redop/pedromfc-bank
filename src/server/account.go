package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// Redefine in32 as money so that we can use make a MarshalJSON method for it.
// We represent the account balance as the actual balance time 100 (e.g. BRL
// 223.15 is represented as 22315). We only support addition/substraction so
// we don't need more decimals than two, for the BRL cents.
type money int32

// Stringify a money value. The last two digits are the cents.
func (num money) String() string {
	return fmt.Sprintf("%d.%d%d", num/100, (num%100)/10, (num%100)%10)
}

func (num money) MarshalJSON() ([]byte, error) {
	return []byte(num.String()), nil
}

// Account entity from accounts table. Fields exported for JSON marshalling,
// except for secret.
type account struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// This is a fake CPF in the format XXX.XXX-XX
	CPF       string    `json:"cpf"`
	secret    string    // Un-exported so won't be JSON-ified for response
	Balance   money     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

// JSON that the client sends to create a new account. Fields exported
// for JSON unmarshalling.
type accountCreateRequest struct {
	Name   string `json:"name"`
	CPF    string `json:"cpf"`
	Secret string `json:"secret"`
}

// Check if the account creation request from the client is valid.
func (accReq *accountCreateRequest) validate() error {
	if len(accReq.Name) > 32 {
		return nameTooLongError
	}

	if len(accReq.Secret) > 32 {
		return pwTooLongError
	}

	if cpfRegex.MatchString(accReq.CPF) {
		return cpfInvalidError
	}

	return nil
}

const startingBalance = "233472"

// Insert a new account into the database, using the values from the client's
// request.
//
// The transaction sequence is:
// - QUERY WHERE CPF = <REQUEST CPF> (the CPF is unique)
// - INSERT ... RETURNING ID
// - QUERY WHERE ID
//
// Then return the account object made from the account we inserted and
// queried back.
//
// This sequence doesn't actually guarantee that the account will be unique by
// the time the transaction tries to commit, but in most cases it will
// show a more useful error to the client. When another concurrent transaction
// wins by inserting a row with the same CPF value, the client will get an
// internal server error. If the account with the same CPF already existed
// before we start the transaction, the client will get a nice error message
// saying the account already exists.
func insertAccount(ctx context.Context,
	accountReq *accountCreateRequest) (*account, error) {

	var err error
	var tx *sql.Tx
	var acc account

	tx, err = db.BeginTx(ctx, &defaultTxOptions)

	if err != nil {
		logger.Print("Error starting tx to insert account")
		return nil, err
	}

	var id int
	var row *sql.Row

	row = tx.QueryRow(`select id from accounts where CPF = $1`, accountReq.CPF)
	err = row.Scan(&id)

	if err == sql.ErrNoRows {
		// We're good, no duplicate currently.
	} else if err == nil {
		rollbackTx(tx)
		return nil, accExistsError
	} else {
		rollbackTx(tx)
		logger.Printf("Error checking for account duplicate")
		return nil, err
	}

	row = tx.QueryRow(
		`insert into accounts (name, cpf, secret, balance, created_at)
		values ($1, $2, $3, $4, current_timestamp at time zone 'UTC')
		returning id`,
		accountReq.Name,
		accountReq.CPF,
		fmt.Sprintf("%x", sha256.Sum256([]byte(accountReq.Secret))),
		startingBalance)

	err = row.Scan(&id)

	if err != nil {
		logger.Printf("Error inserting account")
		rollbackTx(tx)
		return nil, err
	}

	row = tx.QueryRow(
		`select id,name,cpf,secret,balance,
		created_at from accounts where id = $1`, id)

	err = row.Scan(
		&acc.ID, &acc.Name, &acc.CPF,
		&acc.secret, &acc.Balance,
		&acc.CreatedAt)

	if err != nil {
		logger.Printf("Error retrieving inserted account")
		rollbackTx(tx)
		return nil, err
	}

	err = tx.Commit()

	if err != nil {
		logger.Print("Error commiting tx")
		return nil, err
	}

	logger.Printf("Inserted account with id %d", acc.ID)
	return &acc, nil
}

// Handler for creating an account for POST requests at /accounts
func createAccount(rw http.ResponseWriter, req *http.Request) {
	var acc *account
	var accountReq accountCreateRequest

	// About 32 + 10 + 32 = 74 bytes for the values, plus change for json
	// enconding and field names.
	var data, err = readFromReq(req, 256)

	if err != nil {
		respondWithError(rw, err)
		return
	}

	err = json.Unmarshal(data, &accountReq)

	if err != nil {
		respondWithError(rw, cantParseJSONError)
		return
	}

	err = accountReq.validate()

	if err != nil {
		respondWithError(rw, err)
		return
	}

	acc, err = insertAccount(req.Context(), &accountReq)

	if err != nil {
		respondWithError(rw, err)
		return
	}

	jsonResponse, err := json.Marshal(acc)

	if err != nil {
		logger.Printf(
			"Could not marshal account json for response")

		respondWithError(rw, err)
		return
	}

	setJSONEncoding(rw)
	rw.WriteHeader(http.StatusCreated)

	_, err = rw.Write(jsonResponse)

	// A whitespace is allowed at the end of json and it's nicer when
	// curling this serice from the command line.
	if err == nil {
		_, err = rw.Write([]byte("\n"))
	}

	if err != nil {
		logger.Printf("Could not write response: %v", err)
	}
}

// Handler for getting a list of accounts for GET requests at /accounts
func getAccounts(rw http.ResponseWriter, req *http.Request) {

	rows, err := db.Query(
		"select id, name, cpf, balance, created_at from accounts")

	if err != nil {
		respondWithError(rw, err)
		return
	}

	defer rows.Close()

	var acc account

	// Ideally we'd have some form of pagination. For simplicity, allocate a
	// reasonable amount of space, and assume there won't be too many accounts
	// to return to the client.
	var accounts []account = make([]account, 0, 64)

	next_p := rows.Next()

	for next_p {
		err = rows.Scan(&acc.ID, &acc.Name, &acc.CPF,
			&acc.Balance, &acc.CreatedAt)

		if err != nil {
			logger.Printf("error when querying accounts")
			respondWithError(rw, err)
			return
		}

		accounts = append(accounts, acc)

		next_p = rows.Next()
	}

	if err = rows.Err(); err != nil {
		logger.Printf("error when querying accounts")
		respondWithError(rw, err)
		return
	}

	var jsonResponse []byte
	jsonResponse, err = json.Marshal(accounts)

	if err != nil {
		logger.Printf("error when marshalling accounts")
		respondWithError(rw, err)
		return
	}

	setJSONEncoding(rw)

	_, err = rw.Write(jsonResponse)

	if err == nil {
		_, err = rw.Write([]byte("\n"))
	}

	if err != nil {
		logger.Printf("Could not write response: %v", err)
	}
}

// Route requests to /accounts depending on the method (GET or POST)
func handleAccounts(rw http.ResponseWriter, req *http.Request) {

	if req.URL.Path != "/accounts" {
		respondWithError(rw, invalidURLError)
		return
	}

	if req.Method == http.MethodPost {
		createAccount(rw, req)
	} else if req.Method == http.MethodGet {
		getAccounts(rw, req)
	} else {
		respondWithError(rw, invalidMethodError)
	}
}

// Regex to match account balance requests.
var balanceURLRegex *regexp.Regexp = regexp.MustCompile(
	`^/accounts/([0-9]+)/balance$`)

// Handles GET requests at /accounts/<id>/balance
func getAccountBalance(rw http.ResponseWriter, req *http.Request) {
	var err error

	matches := balanceURLRegex.FindStringSubmatch(req.URL.Path)

	if matches == nil || len(matches) != 2 {
		respondWithError(rw, invalidURLError)
		return
	}

	if req.Method != http.MethodGet {
		respondWithError(rw, invalidMethodError)
		return
	}

	// Our db uses ints for the ids, so max 32 bits. The regex already
	// disallows negative numbers.
	id, err := strconv.ParseInt(matches[1], 0, 32)

	if errors.Is(err, strconv.ErrRange) {
		respondWithError(rw, idTooLargeError)
		return
	} else if err != nil {
		// Shouldn't happen since we validated the int from the regex
		logger.Printf("Could not convert url int (???)")
		respondWithError(rw, err)
		return
	}

	logger.Printf("Getting balance for account %d", id)

	row := db.QueryRow("select balance from accounts where id = $1", id)

	var balance money
	err = row.Scan(&balance)

	if err == sql.ErrNoRows {
		respondWithError(rw, noAccountError)
		return
	} else if err != nil {
		respondWithError(rw, err)
		return
	}

	setJSONEncoding(rw)

	_, err = rw.Write([]byte(fmt.Sprintf("{\"id\":%d,\"balance\":%s}\n",
		id, balance.String())))

	if err != nil {
		logger.Printf("Could not write response: %v", err)
	}
}
