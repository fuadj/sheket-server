package models

/*
import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestCreateUserNotExist(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_username).
		WillReturnRows(sqlmock.NewRows(_cols("user_id, username, hashpass")))
	mock.ExpectQuery(
		fmt.Sprintf("insert into %s", TABLE_USER)).
		WithArgs(t_username, t_pass_hash).
		WillReturnRows(
			sqlmock.NewRows(_cols("user_id")).AddRow(t_user_id))
	mock.ExpectCommit()

	u := &User{Username: t_username, HashedPassword: t_pass_hash}
	_, err := store.CreateUser(u)
	if err != nil {
		t.Errorf("CreateUser error '%v'", err)
	}
}

func TestCreateUserNotExistFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_username).
		WillReturnRows(sqlmock.NewRows(_cols("user_id, username, hashpass")))
	mock.ExpectQuery(
		fmt.Sprintf("insert into %s", TABLE_USER)).
		WithArgs(t_username, t_pass_hash).WillReturnError(fmt.Errorf("insert error"))

	mock.ExpectRollback()

	u := &User{Username: t_username, HashedPassword: t_pass_hash}
	_, err := store.CreateUser(u)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestCreateUserExistRollback(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_username).
		WillReturnRows(sqlmock.NewRows(_cols("user_id, username, hashpass")).
			AddRow(t_user_id, t_username, t_pass_hash))
	mock.ExpectRollback()

	u := &User{Username: t_username, HashedPassword: t_pass_hash}
	_, err := store.CreateUser(u)
	if err == nil {
		t.Errorf("expected user already exists error")
	}
}

func TestCreateUserExistFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_username).
		WillReturnError(fmt.Errorf("select error"))
	mock.ExpectRollback()

	u := &User{Username: t_username, HashedPassword: t_pass_hash}
	_, err := store.CreateUser(u)
	if err == nil {
		t.Errorf("expected select error")
	}
}

func TestFindUserByName(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_username).
		WillReturnRows(sqlmock.NewRows(_cols("user_id, username, hashpass")).
			AddRow(t_user_id, t_username, t_pass_hash))

	_, err := store.FindUserByName(t_username)
	if err != nil {
		t.Errorf("FindUserByName error '%v'", err)
	}
}

func TestFindUserByNameFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_username).
		WillReturnError(fmt.Errorf("select error"))

	_, err := store.FindUserByName(t_username)
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestFindUserById(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_user_id).
		WillReturnRows(sqlmock.NewRows(_cols("user_id, username, hashpass")).
			AddRow(t_user_id, t_username, t_pass_hash))

	_, err := store.FindUserById(t_user_id)
	if err != nil {
		t.Errorf("FindUserById error '%v'", err)
	}
}

func TestFindUserByIdFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(t_user_id).
		WillReturnError(fmt.Errorf("select error"))

	_, err := store.FindUserById(t_user_id)
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestGetUserPermission(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(
			sqlmock.NewRows(_cols("company_id, user_id, permission")).
				AddRow(t_company_id, t_user_id, t_permission))

	u := &User{UserId: t_user_id}
	_, err := store.GetUserPermission(u, t_company_id)
	if err != nil {
		t.Errorf("GetUserPermission error '%v'", err)
	}
}

func TestGetUserPermissionFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnError(fmt.Errorf("select error"))

	u := &User{UserId: t_user_id}
	_, err := store.GetUserPermission(u, t_company_id)
	if err == nil {
		t.Errorf("expected an error")
	}
}

/*
func TestSetUserPermissionInsert(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id, t_permission).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err := store.SetUserPermission(p)
	if err != nil {
		t.Errorf("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionUpdate(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(
		sqlmock.NewRows(_cols("permission")).AddRow(t_permission))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(t_permission, t_company_id, t_user_id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err := store.SetUserPermission(p)
	if err != nil {
		t.Errorf("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionSelectFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err := store.SetUserPermission(p)
	if err == nil {
		t.Errorf("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionInsertFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id, t_permission).
		WillReturnError(fmt.Errorf("insert fail"))
	mock.ExpectRollback()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err := store.SetUserPermission(p)
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestSetUserPermissionUpdateFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission")))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(t_permission, t_company_id, t_user_id).
		WillReturnError(fmt.Errorf("update fail"))
	mock.ExpectRollback()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err := store.SetUserPermission(p)
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestSetUserPermissionInTransactionInsert(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id, t_permission).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tnx, err := db.Begin()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err = store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		t.Errorf("SetUserPermissionInTransaction error '%v'", err)
	}
}

func TestSetUserPermissionInTransactionUpdate(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(
		sqlmock.NewRows(_cols("permission")).AddRow(t_permission))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(t_permission, t_company_id, t_user_id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tnx, err := db.Begin()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err = store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		t.Errorf("SetUserPermissionInTransaction error '%v'", err)
	}
}

func TestSetUserPermissionSelectInTransactionFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnError(fmt.Errorf("some error"))

	tnx, err := db.Begin()

	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err = store.SetUserPermissionInTx(tnx, p)
	if err == nil {
		t.Errorf("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionInsertInTransactionFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id, t_permission).
		WillReturnError(fmt.Errorf("insert fail"))

	tnx, err := db.Begin()
	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err = store.SetUserPermissionInTx(tnx, p)
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestSetUserPermissionUpdateInTransactionFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(t_company_id, t_user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission")).
		AddRow(t_permission))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(t_permission, t_company_id, t_user_id).
		WillReturnError(fmt.Errorf("update fail"))

	tnx, err := db.Begin()
	p := &UserPermission{t_company_id, t_user_id, t_permission}
	p, err = store.SetUserPermissionInTx(tnx, p)
	if err == nil {
		t.Errorf("expected an error")
	}
}
*/
