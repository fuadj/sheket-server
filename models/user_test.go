package models

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestCreateUserNotExist(t *testing.T) {
	mock_setup(t, "TestCreateUserNotExist")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(username).
		WillReturnRows(sqlmock.NewRows(_cols("id, username, hashpass")))
	mock.ExpectQuery(
		fmt.Sprintf("insert into %s", TABLE_USER)).
		WithArgs(username, pass_hash).
		WillReturnRows(
		sqlmock.NewRows(_cols("user_id")).AddRow(user_id))
	mock.ExpectCommit()

	u := &User{Username: username, HashedPassword: pass_hash}
	_, err := store.CreateUser(u, password)
	if err != nil {
		_log_err("CreateUser error '%v'", err)
	}
}

func TestCreateUserNotExistFail(t *testing.T) {
	mock_setup(t, "TestCreateUserNotExistFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(username).
		WillReturnRows(sqlmock.NewRows(_cols("id, username, hashpass")))
	mock.ExpectQuery(
		fmt.Sprintf("insert into %s", TABLE_USER)).
		WithArgs(username, pass_hash).WillReturnError(fmt.Errorf("insert error"))

	mock.ExpectRollback()

	u := &User{Username: username, HashedPassword: pass_hash}
	_, err := store.CreateUser(u, password)
	if err == nil {
		_log_err("expected error")
	}
}

func TestCreateUserExistRollback(t *testing.T) {
	mock_setup(t, "TestCreateUserExistRollback")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(username).
		WillReturnRows(sqlmock.NewRows(_cols("id, username, hashpass")).
		AddRow(user_id, username, pass_hash))
	mock.ExpectRollback()

	u := &User{Username: username, HashedPassword: pass_hash}
	_, err := store.CreateUser(u, password)
	if err == nil {
		_log_err("expected user already exists error")
	}
}

func TestCreateUserExistFail(t *testing.T) {
	mock_setup(t, "TestCreateUserExistFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(username).
		WillReturnError(fmt.Errorf("select error"))
	mock.ExpectRollback()

	u := &User{Username: username, HashedPassword: pass_hash}
	_, err := store.CreateUser(u, password)
	if err == nil {
		_log_err("expected select error")
	}
}

func TestFindUserByName(t *testing.T) {
	mock_setup(t, "TestFindUserByName")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(username).
		WillReturnRows(sqlmock.NewRows(_cols("id, username, hashpass")).
		AddRow(user_id, username, pass_hash))

	_, err := store.FindUserByName(username)
	if err != nil {
		_log_err("FindUserByName error '%v'", err)
	}
}

func TestFindUserByNameFail(t *testing.T) {
	mock_setup(t, "TestFindUserByNameFail")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
		WithArgs(username).
		WillReturnError(fmt.Errorf("select error"))

	_, err := store.FindUserByName(username)
	if err == nil {
		_log_err("expected an error")
	}
}

func TestFindUserById(t *testing.T) {
	mock_setup(t, "TestFindUserById")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
	WithArgs(user_id).
	WillReturnRows(sqlmock.NewRows(_cols("id, username, hashpass")).
	AddRow(user_id, username, pass_hash))

	_, err := store.FindUserById(user_id)
	if err != nil {
		_log_err("FindUserById error '%v'", err)
	}
}

func TestFindUserByIdFail(t *testing.T) {
	mock_setup(t, "TestFindUserByIdFail")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_USER)).
	WithArgs(user_id).
	WillReturnError(fmt.Errorf("select error"))

	_, err := store.FindUserById(user_id)
	if err == nil {
		_log_err("expected an error")
	}
}

func TestGetUserPermission(t *testing.T) {
	mock_setup(t, "TestGetUserPermission")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(
		sqlmock.NewRows(_cols("company_id, user_id, permission_type, branch_id")).
			AddRow(company_id, user_id, permission_type, branch_id))

	p := &UserPermission{CompanyId:company_id, UserId:user_id,
		PermissionType:permission_type, BranchId:branch_id}
	p, err := store.GetUserPermission(p)
	if err != nil {
		_log_err("GetUserPermission error '%v'", err)
	}
}

func TestGetUserPermissionFail(t *testing.T) {
	mock_setup(t, "TestGetUserPermissionFail")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
	WithArgs(company_id, user_id).
	WillReturnError(fmt.Errorf("select error"))

	p := &UserPermission{CompanyId:company_id, UserId:user_id,
		PermissionType:permission_type, BranchId:branch_id}
	_, err := store.GetUserPermission(p)
	if err == nil {
		_log_err("expected an error")
	}
}

func TestSetUserPermissionInsert(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionInsert")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission_type")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id, permission_type, branch_id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err := store.SetUserPermission(p)
	if err != nil {
		_log_err("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionUpdate(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionUpdate")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(
		sqlmock.NewRows(_cols("permission_type")).AddRow(permission_type))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(permission_type, branch_id, company_id, user_id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err := store.SetUserPermission(p)
	if err != nil {
		_log_err("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionSelectFail(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionSelectFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err := store.SetUserPermission(p)
	if err == nil {
		_log_err("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionInsertFail(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionInsertFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission_type")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id, permission_type, branch_id).
		WillReturnError(fmt.Errorf("insert fail"))
	mock.ExpectRollback()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err := store.SetUserPermission(p)
	if err == nil {
		_log_err("expected an error")
	}
}

func TestSetUserPermissionUpdateFail(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionUpdateFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission_type")))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(permission_type, branch_id, company_id, user_id).
		WillReturnError(fmt.Errorf("update fail"))
	mock.ExpectRollback()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err := store.SetUserPermission(p)
	if err == nil {
		_log_err("expected an error")
	}
}

func TestSetUserPermissionInTransactionInsert(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionInTransactionInsert")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission_type")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id, permission_type, branch_id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tnx, err := db.Begin()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err = store.SetUserPermissionInTransaction(tnx, p)
	if err != nil {
		_log_err("SetUserPermissionInTransaction error '%v'", err)
	}
}

func TestSetUserPermissionInTransactionUpdate(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionInTransactionUpdate")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(
		sqlmock.NewRows(_cols("permission_type")).AddRow(permission_type))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(permission_type, branch_id, company_id, user_id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tnx, err := db.Begin()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err = store.SetUserPermissionInTransaction(tnx, p)
	if err != nil {
		_log_err("SetUserPermissionInTransaction error '%v'", err)
	}
}

func TestSetUserPermissionSelectInTransactionFail(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionSelectInTransactionFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnError(fmt.Errorf("some error"))

	tnx, err := db.Begin()

	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err = store.SetUserPermissionInTransaction(tnx, p)
	if err == nil {
		_log_err("SetUserPermission error '%v'", err)
	}
}

func TestSetUserPermissionInsertInTransactionFail(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionInsertInTransactionFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission_type")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id, permission_type, branch_id).
		WillReturnError(fmt.Errorf("insert fail"))

	tnx, err := db.Begin()
	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err = store.SetUserPermissionInTransaction(tnx, p)
	if err == nil {
		_log_err("expected an error")
	}
}

func TestSetUserPermissionUpdateInTransactionFail(t *testing.T) {
	mock_setup(t, "TestSetUserPermissionUpdateInTransactionFail")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_U_PERMISSION)).
		WithArgs(company_id, user_id).
		WillReturnRows(sqlmock.NewRows(_cols("permission_type")).
			AddRow(permission_type))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_U_PERMISSION)).
		WithArgs(permission_type, branch_id, company_id, user_id).
		WillReturnError(fmt.Errorf("update fail"))

	tnx, err := db.Begin()
	p := &UserPermission{company_id, user_id, permission_type, branch_id}
	p, err = store.SetUserPermissionInTransaction(tnx, p)
	if err == nil {
		_log_err("expected an error")
	}
}
