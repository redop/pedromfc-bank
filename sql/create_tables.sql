DROP DATABASE pedro_bank;
CREATE DATABASE pedro_bank;

\c pedro_bank

ALTER DATABASE pedro_bank SET default_transaction_isolation TO
"repeatable read";

CREATE TABLE accounts (
    -- Start at 1 to avoid errors when unmarshalling json with no id field,
    -- which would become 0.
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY (START 1),
    name VARCHAR(32) NOT NULL,
    -- XXX.XXX-XX: a "fake" CPF with 8 digits (to avoid accidentaly using a
    -- real person's CPF)
    cpf CHAR(10) NOT NULL UNIQUE,
    -- 32 bytes (256 bits) for a sha256 hash of the password, represented as
    -- 64 hex digits
    secret CHAR(64) NOT NULL,
    -- We represent the balance as integers where the last two digits are the
    -- BRL cents. We don't need more precision sicne we only add/substract
    -- from the account balance.
    balance INTEGER NOT NULL,
    -- No time zone, store always as UTC
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE transfers (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY (START 1),
    origin_id INTEGER NOT NULL REFERENCES accounts (id),
    destination_id INTEGER NOT NULL REFERENCES accounts (id),
    amount INTEGER NOT NULL,
    -- No time zone, store always as UTC
    created_at TIMESTAMP NOT NULL
);