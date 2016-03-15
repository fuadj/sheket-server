package models

import (
	"database/sql"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"strings"
	"testing"
)

const (
	company_id      = int64(1)
	company_name    = "test company"
	company_contact = "0912646275"
	username        = "test user"
	user_id         = int64(12312)
	password        = "abcd abcd"
	pass_hash       = "xxkkadlkjaf"
	branch_name     = "test branch"
	permission_type = 4 //	means smth
	category_id     = 2
	branch_location = "mexico"
	branch_id       = int64(10)
	item_id         = int64(88)
	quantity        = 8.8
	item_location   = "A10"
	transaction_id  = 748
	date            = 87123
)

var (
	fn_name string
	ts      *testing.T
	db      *sql.DB
	mock    sqlmock.Sqlmock
	store   ShDataStore
)

func mock_setup(t *testing.T, f_name string) {
	ts = t
	fn_name = f_name
	var err error
	db, mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when testing db", err)
	}
	store = NewShDataStore(db)
}

func mock_teardown() {
	if err := mock.ExpectationsWereMet(); err != nil {
		_log_err("Expectation not met %v", err)
	}
	db.Close()
}

func _log_err(format string, args ...interface{}) {
	ts.Errorf(fmt.Sprintf("%s %s", fn_name, format), args...)
}

// given CSV format string, returns array of elems
// useful in {@code sqlmock.NewRows}
func _cols(s string) []string {
	subs := strings.Split(s, ",")
	for i := 0; i < len(subs); i++ {
		subs[i] = strings.TrimSpace(subs[i])
	}
	return subs
}
