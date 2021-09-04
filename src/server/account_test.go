package server

import (
	"testing"
)

func TestAccountValidation(t *testing.T) {
	var accReq accountCreateRequest
	accReq.Name = "John Doe"
	accReq.CPF = "222.111-11"
	accReq.Secret = "toto"

	if accReq.validate() != nil {
		t.Fail()
	}

	var badAccReq accountCreateRequest = accReq

	badCPFs := []string{"222.111-122", "what", "22211111", "", "222.111.11"}

	for _, badCPF := range badCPFs {
		badAccReq.CPF = badCPF
		if badAccReq.validate() != cpfInvalidError {
			t.Error(badAccReq)
		}
	}

	badAccReq = accReq

	badAccReq.Name = "John With a Very Long Name Very Long Indeed"

	if badAccReq.validate() != nameTooLongError {
		t.Error(badAccReq)
	}

	badAccReq = accReq

	badAccReq.Secret = "This is a very long password, very long indeed"

	if badAccReq.validate() != pwTooLongError {
		t.Error(badAccReq)
	}
}
