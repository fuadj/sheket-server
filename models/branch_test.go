package models

import "testing"
import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"strings"
)

func TestCreateBranch(t *testing.T) {
	mock_setup(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_BRANCH)).
		WithArgs(t_company_id, t_branch_name, t_branch_location).
		WillReturnRows(sqlmock.NewRows(_cols("branch_id")).AddRow(t_branch_id))

	branch := &ShBranch{t_company_id, 1, t_branch_name, t_branch_location}

	branch, err := store.CreateBranch(branch)
	if err != nil {
		t.Errorf("Branch creation failed '%v'", err)
	} else if branch.BranchId != t_branch_id {
		t.Errorf("Expected brach with id:%d", t_branch_id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Expectation not met %v", err)
	}
}

func TestCreateBranchFail(t *testing.T) {
	mock_setup(t)
	defer db.Close()

	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_BRANCH)).
		WithArgs(t_company_id, t_branch_name, t_branch_location).
		WillReturnError(fmt.Errorf("insert error"))

	branch := &ShBranch{t_company_id, 1, t_branch_name, t_branch_location}

	branch, err := store.CreateBranch(branch)
	if err == nil {
		t.Errorf("error should have returned")
	}
}

func TestUpdateBranch(t *testing.T) {
	mock_setup(t)

	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_BRANCH)).
		WithArgs(t_branch_name, t_branch_location, t_branch_id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tnx, _ := db.Begin()
	branch := &ShBranch{t_company_id, 1, t_branch_name, t_branch_location}

	branch.BranchId = t_branch_id
	_, err := store.UpdateBranchInTx(tnx, branch)
	if err != nil {
		t.Errorf("update branch failed %v", err)
	}
}

func TestUpdateBranchFail(t *testing.T) {
	mock_setup(t)

	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_BRANCH)).
		WithArgs(t_branch_name, t_branch_location, t_branch_id).
		WillReturnError(fmt.Errorf("update error"))

	tnx, _ := db.Begin()
	branch := &ShBranch{t_company_id, 1, t_branch_name, t_branch_location}

	branch.BranchId = t_branch_id
	_, err := store.UpdateBranchInTx(tnx, branch)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGetBranch(t *testing.T) {
	mock_setup(t)
	defer db.Close()

	get_rows := sqlmock.NewRows(_cols("company_id,branch_id,branch_name,location")).
		AddRow(t_company_id, t_branch_id, t_branch_name, t_branch_location)
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH)).
		WithArgs(t_branch_id).
		WillReturnRows(get_rows)

	branch, err := store.GetBranchById(t_branch_id)
	if err != nil {
		t.Errorf("GetBranchById failed '%v'", err)
	} else if branch.BranchId != t_branch_id {
		t.Errorf("Expected brach with id:%d", t_branch_id)
	}
}

func TestGetBranchFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH)).
		WithArgs(t_branch_id).
		WillReturnError(fmt.Errorf("invalid branch id error"))

	_, err := store.GetBranchById(t_branch_id)
	if err == nil {
		t.Errorf("no branch created, should have failed '%v'", err)
	}
}

func TestAddBranchItemInsert(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_branch_id, t_item_id).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_company_id, t_branch_id, t_item_id,
		t_quantity, t_item_location).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	item := &ShBranchItem{t_company_id, t_branch_id,
		t_item_id, t_quantity, t_item_location}
	_, err := store.AddItemToBranch(item)

	if err != nil {
		t.Errorf("AddItemToBranch failed '%v'", err)
	}
}

func TestAddBranchItemUpdate(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_branch_id, t_item_id).
		WillReturnRows(
		sqlmock.NewRows(_cols("item_id")).AddRow(t_item_id))

	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_quantity, t_item_location, t_branch_id, t_item_id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	item := &ShBranchItem{t_company_id, t_branch_id,
		t_item_id, t_quantity, t_item_location}
	_, err := store.AddItemToBranch(item)

	if err != nil {
		t.Errorf("AddItemToBranch failed '%v'", err)
	}
}

func TestAddBranchItemInsertRollback(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_branch_id, t_item_id).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")))
	mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_company_id, t_branch_id, t_item_id,
		t_quantity, t_item_location).
		WillReturnError(fmt.Errorf("Insert error"))
	mock.ExpectRollback()

	item := &ShBranchItem{t_company_id, t_branch_id,
		t_item_id, t_quantity, t_item_location}
	_, err := store.AddItemToBranch(item)

	if err == nil {
		t.Errorf("AddItemToBranch expected error")
	}
}

func TestAddBranchItemUpdateRollback(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_branch_id, t_item_id).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")).
		AddRow(t_item_id))
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_quantity, t_item_location, t_branch_id, t_item_id).
		WillReturnError(fmt.Errorf("update error"))
	mock.ExpectRollback()

	item := &ShBranchItem{t_company_id, t_branch_id,
		t_item_id, t_quantity, t_item_location}
	_, err := store.AddItemToBranch(item)

	if err == nil {
		t.Errorf("AddItemToBranch expected error")
	}
}

func TestUpdateItemInBranch(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectExec(
		fmt.Sprintf("update %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_quantity, t_item_location, t_branch_id, t_item_id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	item := &ShBranchItem{t_company_id, t_branch_id,
		t_item_id, t_quantity, t_item_location}
	tnx, _ := db.Begin()
	_, err := store.UpdateBranchItemInTx(tnx, item)
	if err != nil {
		t.Errorf("UpdateItemInBranch error '%v'", err)
	}
}

func TestUpdateItemInBranchFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectExec(
		fmt.Sprintf("update %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_quantity, t_item_location, t_branch_id, t_item_id).
		WillReturnError(fmt.Errorf("update error"))
	item := &ShBranchItem{t_company_id, t_branch_id,
		t_item_id, t_quantity, t_item_location}
	tnx, _ := db.Begin()
	_, err := store.UpdateBranchItemInTx(tnx, item)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGetItemsInBranch(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	rs := sqlmock.NewRows(strings.Split("company_id,branch_id,item_id,quantity,item_location", ",")).
		AddRow(t_company_id, t_branch_id, t_item_id, t_quantity, t_item_location)
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_branch_id).
		WillReturnRows(rs)

	items, err := store.GetItemsInBranch(t_branch_id)
	if err != nil {
		t.Errorf("GetItemsInBranch err '%v'", err)
	}
	if items == nil || len(items) == 0 {
		t.Errorf("No item in branch returned")
	}
	if items[0].ItemId != t_item_id {
		t.Errorf("returned item not the item")
	}
}

func TestGetItemsInBranchFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_branch_id).
		WillReturnError(fmt.Errorf("select error"))

	items, err := store.GetItemsInBranch(t_branch_id)
	if err == nil {
		t.Errorf("GetItemsInBranch should have returned error")
	}
	if items != nil {
		t.Errorf("the items result should have been nil")
	}
}

func TestGetItemsInAllCompanyBranches(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	rs := sqlmock.NewRows(strings.Split("company_id,branch_id,item_id,quantity,item_location", ",")).
		AddRow(t_company_id, t_branch_id, t_item_id, t_quantity, t_item_location)
	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_company_id).
		WillReturnRows(rs)

	items, err := store.GetItemsInAllCompanyBranches(t_company_id)
	if err != nil {
		t.Errorf("err '%v'", err)
	}
	if items == nil || len(items) == 0 {
		t.Errorf("No item in all branches returned")
	}
	if items[0].ItemId != t_item_id {
		t.Errorf("returned item not the item")
	}
}

func TestGetItemsInAllCompanyBranchesFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_BRANCH_ITEM)).
		WithArgs(t_branch_id).
		WillReturnError(fmt.Errorf("select error"))

	items, err := store.GetItemsInAllCompanyBranches(t_branch_id)
	if err == nil {
		t.Errorf("should have returned error")
	}
	if items != nil {
		t.Errorf("the items result should have been nil")
	}
}
