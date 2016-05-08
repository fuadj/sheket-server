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
	t_ctrl       *gomock.Controller
	t_mock       *models.ComposableShStoreMock
	t_save_store models.ShStore
	t_user       *models.User

	t_tnx_setup bool = false
	t_tnx       *sql.Tx
	t_db        *sql.DB
	t_db_mock   sqlmock.Sqlmock
)

var save_getter func(*http.Request) (*models.User, error)

func setup_user(t *testing.T, user_id int64) {
	save_getter = currentUserGetter
	t_user = &models.User{UserId: user_id}
	currentUserGetter = func(*http.Request) (*models.User, error) {
		return t_user, nil
	}
}

func teardown_user() {
	currentUserGetter = save_getter
}

func setup_store(t *testing.T) {
	t_save_store = Store
	t_ctrl = gomock.NewController(t)
	t_mock = models.NewComposableShStoreMock(t_ctrl)
	Store = t_mock
}

func teardown_store() {
	t_ctrl.Finish()
	Store = t_save_store
}

func setup_tnx(t *testing.T) {
	t_tnx_setup = true
	var err error
	t_db, t_db_mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when testing db", err)
	}

	t_db_mock.ExpectBegin()
	t_tnx, _ = t_db.Begin()
}

func teardown_tnx() {
	if t_tnx_setup {
		t_db.Close()
	}
	t_tnx_setup = false
}
