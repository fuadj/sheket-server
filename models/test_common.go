package models

import (
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"strings"
	"testing"
)

const (
	t_company_id      = int64(1)
	t_company_name    = "test company"
	t_company_contact = "0912646275"
	t_username        = "test user"
	t_user_id         = int64(12312)
	t_password        = "abcd abcd"
	t_pass_hash       = "xxkkadlkjaf"
	t_branch_name     = "test branch"
	t_permission      = "{}"
	t_category_id     = 2
	t_branch_location = "mexico"
	t_branch_id       = int64(10)
	t_item_id         = int64(88)
	t_quantity        = 8.8
	t_item_location   = "A10"
	t_date            = 87123
)

var (
	ts    *testing.T
	db    *sql.DB
	mock  sqlmock.Sqlmock
	store ShStore
)

func mock_setup(t *testing.T) {
	ts = t
	var err error
	db, mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when testing db", err)
	}
	store = NewShStore(db)
}

func mock_teardown() {
	if err := mock.ExpectationsWereMet(); err != nil {
		ts.Errorf("Expectation not met %v", err)
	}
	db.Close()
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
