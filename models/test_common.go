package models
import (
	"testing"
	"fmt"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"database/sql"
)

var fn_name string
var ts *testing.T
var db *sql.DB
var mock sqlmock.Sqlmock
var store ShDataStore

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
		log_err("Expectation not met %v", err)
	}
	db.Close()
}

func log_err(format string, args ...interface{}) {
	ts.Errorf(fmt.Sprintf("%s %s", fn_name, format), args...)
}
