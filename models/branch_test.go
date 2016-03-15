package models

import "testing"
import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"strings"
)


func TestCreateBranch(t *testing.T) {
	mock_setup(t, "TestCreateBranch")
	defer db.Close()

	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_BRANCH)).
		WithArgs(company_id, branch_name, branch_location).
		WillReturnRows(sqlmock.NewRows(_cols("branch_id")).AddRow(branch_id))

	branch := &ShBranch{company_id, 1, branch_name, branch_location}

	branch, err := store.CreateBranch(branch)
	if err != nil {
		_log_err("Branch creation failed '%v'", err)
	} else if branch.BranchId != branch_id {
		_log_err("Expected brach with id:%d", branch_id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		_log_err("Expectation not met %v", err)
	}
}

func TestCreateBranchFail(t *testing.T) {
	mock_setup(t, "TestCreateBranchFail")
	defer db.Close()

	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_BRANCH)).
		WithArgs(company_id, branch_name, branch_location).
		WillReturnError(fmt.Errorf("insert error"))

	branch := &ShBranch{company_id, 1, branch_name, branch_location}

	branch, err := store.CreateBranch(branch)
	if err == nil {
		_log_err("error should have returned")
	}
}

func TestGetBranch(t *testing.T) {
	mock_setup(t, "TestGetBranch")
	defer db.Close()

	get_rows := sqlmock.NewRows(_cols("company_id,branch_id,branch_name,location")).
		AddRow(company_id, branch_id, branch_name, branch_location)
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH)).
		WithArgs(branch_id).
		WillReturnRows(get_rows)

	branch, err := store.GetBranchById(branch_id)
	if err != nil {
		_log_err("GetBranchById failed '%v'", err)
	} else if branch.BranchId != branch_id {
		_log_err("Expected brach with id:%d", branch_id)
	}
}

func TestGetBranchFail(t *testing.T) {
	mock_setup(t, "TestGetBranchFail")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH)).
		WithArgs(branch_id).
		WillReturnError(fmt.Errorf("invalid branch id error"))

	_, err := store.GetBranchById(branch_id)
	if err == nil {
		_log_err("no branch created, should have failed '%v'", err)
	}
}

func TestAddBranchItemInsert(t *testing.T) {
	mock_setup(t, "TestAddBranchItemInsert")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(branch_id, item_id).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_BRANCH_ITEM)).
		WithArgs(company_id, branch_id, item_id,
		quantity, item_location).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	item := &ShBranchItem{company_id, branch_id,
		item_id, quantity, item_location}
	_, err := store.AddItemToBranch(item)

	if err != nil {
		_log_err("AddItemToBranch failed '%v'", err)
	}
}

func TestAddBranchItemUpdate(t *testing.T) {
	mock_setup(t, "TestAddBranchItemUpdate")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(branch_id, item_id).
		WillReturnRows(
		sqlmock.NewRows(_cols("item_id")).AddRow(item_id))

	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_BRANCH_ITEM)).
		WithArgs(quantity, item_location, branch_id, item_id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	item := &ShBranchItem{company_id, branch_id,
		item_id, quantity, item_location}
	_, err := store.AddItemToBranch(item)

	if err != nil {
		_log_err("AddItemToBranch failed '%v'", err)
	}
}

func TestAddBranchItemInsertRollback(t *testing.T) {
	mock_setup(t, "TestAddBranchItemInsertRollback")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(branch_id, item_id).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_BRANCH_ITEM)).
		WithArgs(company_id, branch_id, item_id,
		quantity, item_location).
		WillReturnError(fmt.Errorf("Insert error"))
	mock.ExpectRollback()

	item := &ShBranchItem{company_id, branch_id,
		item_id, quantity, item_location}
	_, err := store.AddItemToBranch(item)

	if err == nil {
		_log_err("AddItemToBranch expected error")
	}
}

func TestAddBranchItemUpdateRollback(t *testing.T) {
	mock_setup(t, "TestAddBranchItemUpdateRollback")
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(branch_id, item_id).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")).
			AddRow(item_id))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_BRANCH_ITEM)).
		WithArgs(quantity, item_location, branch_id, item_id).
		WillReturnError(fmt.Errorf("update error"))
	mock.ExpectRollback()

	item := &ShBranchItem{company_id, branch_id,
		item_id, quantity, item_location}
	_, err := store.AddItemToBranch(item)

	if err == nil {
		_log_err("AddItemToBranch expected error")
	}
}

func TestGetItemsInBranch(t *testing.T) {
	mock_setup(t, "TestGetItemsInBranch")
	defer mock_teardown()

	rs := sqlmock.NewRows(strings.Split("company_id,branch_id,item_id,quantity,item_location", ",")).
		AddRow(company_id, branch_id, item_id, quantity, item_location)
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(branch_id).
		WillReturnRows(rs)

	items, err := store.GetItemsInBranch(branch_id)
	if err != nil {
		_log_err("GetItemsInBranch err '%v'", err)
	}
	if items == nil || len(items) == 0 {
		_log_err("No item in branch returned")
	}
	if items[0].ItemId != item_id {
		_log_err("returned item not the item")
	}
}

func TestGetItemsInBranchFail(t *testing.T) {
	mock_setup(t, "TestGetItemsInBranchFail")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(branch_id).
		WillReturnError(fmt.Errorf("select error"))

	items, err := store.GetItemsInBranch(branch_id)
	if err == nil {
		_log_err("GetItemsInBranch should have returned error")
	}
	if items != nil {
		_log_err("the items result should have been nil")
	}
}

func TestGetItemsInAllCompanyBranches(t *testing.T) {
	mock_setup(t, "TestGetItemsInAllCompanyBranches")
	defer mock_teardown()

	rs := sqlmock.NewRows(strings.Split("company_id,branch_id,item_id,quantity,item_location", ",")).
		AddRow(company_id, branch_id, item_id, quantity, item_location)
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(company_id).
		WillReturnRows(rs)

	items, err := store.GetItemsInAllCompanyBranches(company_id)
	if err != nil {
		_log_err("err '%v'", err)
	}
	if items == nil || len(items) == 0 {
		_log_err("No item in all branches returned")
	}
	if items[0].ItemId != item_id {
		_log_err("returned item not the item")
	}
}

func TestGetItemsInAllCompanyBranchesFail(t *testing.T) {
	mock_setup(t, "TestGetItemsInBranchFail")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(branch_id).
		WillReturnError(fmt.Errorf("select error"))

	items, err := store.GetItemsInAllCompanyBranches(branch_id)
	if err == nil {
		_log_err("should have returned error")
	}
	if items != nil {
		_log_err("the items result should have been nil")
	}
}
