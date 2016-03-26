package controller

import (
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"net/http"
	"sheket/server/models"
	"testing"
)

var (
	ctrl       *gomock.Controller
	mock       *models.ComposableShStoreMock
	save_store models.ShStore
	user       *models.User

	tnx_setup bool = false
	tnx       *sql.Tx
	db        *sql.DB
	db_mock   sqlmock.Sqlmock
)

var save_getter func(*http.Request) (*models.User, error)

func setup_user(t *testing.T, user_id int64) {
	save_getter = currentUserGetter
	user = &models.User{UserId: user_id}
	currentUserGetter = func(*http.Request) (*models.User, error) {
		return user, nil
	}
}

func teardown_user() {
	currentUserGetter = save_getter
}

func setup_store(t *testing.T) {
	save_store = Store
	ctrl = gomock.NewController(t)
	mock = models.NewComposableShStoreMock(ctrl)
	Store = mock
}

func teardown_store() {
	ctrl.Finish()
	Store = save_store
}

func setup_tnx(t *testing.T) {
	tnx_setup = true
	var err error
	db, db_mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when testing db", err)
	}

	db_mock.ExpectBegin()
	tnx, _ = db.Begin()
}

func teardown_tnx() {
	if tnx_setup {
		db.Close()
	}
	tnx_setup = false
}
