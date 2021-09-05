package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

// JSON that the client sends to create a new account. Fields exported
// for JSON unmarshalling.
type transferRequest struct {
	DestinationID int   `json:"account_destination_id"`
	Amount        money `json:"amount"`
}

// Transfer entity
type transfer struct {
	ID            int       `json:"id"`
	OriginID      int       `json:"account_origin_id"`
	DestinationID int       `json:"account_destination_id"`
	Amount        money     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

// Insert a new transfer
func insertTransfer(
	ctx context.Context,
	origID int,
	destID int,
	amount money) (*transfer, error) {

	var err error
	var tx *sql.Tx
	var transf transfer
	var row *sql.Row
	var origBalance, destBalance money

	tx, err = DB.BeginTx(ctx, &defaultTxOptions)

	if err != nil {
		logger.Print("Error starting tx to insert transfer")
		return nil, err
	}

	// We lock the account rows with FOR UPDATE in case they get modified
	// by another transaction.
	//
	// Because Postgres uses MVCC, the balance we get might not actually be
	// correct when the transaction commits, if another transaction updated
	// it, unless we lock the rows explicitly.
	//
	// E.g., if another transfer happens concurrently and we didn't lock the
	// account row, we could end up setting the final balances from this
	// transfer and lose the update from the concurrent trasnfer.
	accQuery := `select balance from accounts where id = $1 for update`

	row = tx.QueryRow(accQuery, origID)
	err = row.Scan(&origBalance)

	// We need to check that the origin and destination accounts
	// actually exist in the DB. The login map is in-memory and not
	// synchronized with the DB, so if someone were to add a feature
	// to remove accounts in the future, we shouls handle this case.
	if err == sql.ErrNoRows {
		rollbackTx(tx)
		return nil, noOrigAccountError
	} else if err != nil {
		rollbackTx(tx)
		return nil, err
	}

	row = tx.QueryRow(accQuery, destID)
	err = row.Scan(&destBalance)

	if err == sql.ErrNoRows {
		rollbackTx(tx)
		return nil, noDestAccountError
	} else if err != nil {
		rollbackTx(tx)
		return nil, err
	}

	if origBalance < amount {
		rollbackTx(tx)
		return nil, insufficientFundsError
	}

	// We represent our money as an int, that is, the actual money * 100,
	// so we don't have to worry about handling decimal parts, just presenting
	// correctly to the user.
	origBalance = origBalance - amount

	bigDestBalance := big.NewInt(int64(destBalance))
	bigDestBalance.Add(bigDestBalance, big.NewInt(int64(amount)))

	// We used a signed int, so the new balance has to be representable in 31
	// bits. We don't have negatie balances.
	if bigDestBalance.BitLen() > 31 {
		rollbackTx(tx)
		return nil, amountTooLargeError
	}

	// We know from the previous check that the new balance fits.
	destBalance = money(bigDestBalance.Int64())

	accQuery = `update accounts set balance = $1 where id = $2`

	var res sql.Result

	// Update origin
	res, err = tx.Exec(accQuery, origBalance, origID)

	if err != nil {
		rollbackTx(tx)
		return nil, err
	} else {
		var rowsAffected int64
		rowsAffected, err = res.RowsAffected()

		if err != nil {
			rollbackTx(tx)
			return nil, err
		} else if rowsAffected != 1 {
			rollbackTx(tx)
			return nil, fmt.Errorf("unexpected number of affected rows")
		}
	}

	// Update destination
	res, err = tx.Exec(accQuery, destBalance, destID)

	if err != nil {
		rollbackTx(tx)
		return nil, err
	} else {
		var rowsAffected int64
		rowsAffected, err = res.RowsAffected()

		if err != nil {
			rollbackTx(tx)
			return nil, err
		} else if rowsAffected != 1 {
			rollbackTx(tx)
			return nil, fmt.Errorf("unexpected number of affected rows")
		}
	}

	// Now, insert the actual transfer record
	var id int

	row = tx.QueryRow(
		`insert into transfers (origin_id, destination_id, amount, created_at)
		values ($1, $2, $3, current_timestamp at time zone 'UTC')
		returning id`,
		origID,
		destID,
		amount)

	err = row.Scan(&id)

	if err != nil {
		rollbackTx(tx)
		return nil, err
	}

	row = tx.QueryRow(
		`select id, origin_id, destination_id, amount, created_at
		created_at from transfers where id = $1`, id)

	err = row.Scan(
		&transf.ID, &transf.OriginID, &transf.DestinationID,
		&transf.Amount, &transf.CreatedAt)

	if err != nil {
		rollbackTx(tx)
		return nil, err
	}

	err = tx.Commit()

	if err != nil {
		logger.Print("Error commiting tx")
		return nil, err
	}

	return &transf, nil
}

// Handler for POST at /transfers. Gets the origin id from the token, if any,
// and the destination id from the request.
func transferHandler(rw http.ResponseWriter, req *http.Request) {

	if req.URL.Path != "/transfers" {
		respondWithError(rw, invalidURLError)
		return
	}

	if req.Method != http.MethodPost {
		respondWithError(rw, invalidMethodError)
		return
	}

	token := req.Header.Get("Authorization")

	if token == "" {
		respondWithError(rw, noTokenError)
		return
	}

	id, err := getUserByToken(token, true)

	if err != nil {
		respondWithError(rw, err)
		return
	}

	var data []byte
	data, err = readFromReq(req, 128)

	if err != nil {
		respondWithError(rw, err)
		return
	}

	var transferReq transferRequest

	err = json.Unmarshal(data, &transferReq)

	var publicError *publicJSONError
	if errors.As(err, &publicError) {
		respondWithError(rw, publicError)
		return
	} else if err != nil {
		respondWithError(rw, cantParseJSONError)
		return
	}

	// This can also happen if the user doesn't specify the amount in the
	// request JSON. Unfortunately we can't tell the stdlib json functions
	// to require a given field.
	if transferReq.Amount == 0 {
		respondWithError(rw, zeroAmountError)
		return
	}

	// Same here, if no destination_id field in the JSON, it will be 0.
	// Note that our db assigns account ids starting at 1.
	if transferReq.DestinationID == 0 {
		respondWithError(rw, badDestinationIdError)
		return
	}

	var transf *transfer
	transf, err = insertTransfer(req.Context(), id, transferReq.DestinationID,
		transferReq.Amount)

	if err != nil {
		respondWithError(rw, err)
		return
	}

	var jsonResponse []byte
	jsonResponse, err = json.Marshal(transf)

	if err != nil {
		logger.Printf(
			"Could not marshal transf json for response")

		respondWithError(rw, err)
		return
	}

	rw.WriteHeader(http.StatusCreated)
	setJSONEncoding(rw)

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
