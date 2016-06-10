package models

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

const (
	item_name         = "'test item name'"
	item_model        = "1992"
	item_part_number  = "00122"
	item_bar_code     = "0123456789"
	item_has_bar_code = true
	item_manual_code  = ""
)

func dummyTestItem() *ShItem {
	i := &ShItem{CompanyId: t_company_id,
		Name: item_name, ModelYear: item_model,
		PartNumber: item_part_number, BarCode: item_bar_code,
		HasBarCode: item_has_bar_code, ItemCode: item_manual_code}
	return i
}

func TestCreateInventoryItem(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_INVENTORY_ITEM)).
		WithArgs(t_company_id, item_name, item_model,
			item_part_number, item_bar_code, item_has_bar_code, item_manual_code).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")).AddRow(t_item_id))

	item, err := store.CreateItem(dummyTestItem())
	if err != nil {
		t.Errorf("CreateItem error '%v'", err)
	} else if item.ItemId != t_item_id {
		t.Errorf("Not the expected item")
	}
}

func TestCreateInventoryItemFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_INVENTORY_ITEM)).
		WithArgs(t_company_id, item_name, item_model,
			item_part_number, item_bar_code, item_has_bar_code, item_manual_code).
		WillReturnError(fmt.Errorf("insert error"))

	_, err := store.CreateItem(dummyTestItem())
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestUpdateInventoryItem(t *testing.T) {
	mock_setup(t)

	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_INVENTORY_ITEM)).
		WithArgs(item_name, item_model,
			item_part_number, item_bar_code, item_has_bar_code, item_manual_code,
			t_item_id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tnx, _ := db.Begin()
	item := dummyTestItem()
	item.ItemId = t_item_id
	_, err := store.UpdateItemInTx(tnx, item)
	if err != nil {
		t.Errorf("update item failed %v", err)
	}
}

func TestUpdateInventoryItemFail(t *testing.T) {
	mock_setup(t)

	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectExec(fmt.Sprintf("update %s", TABLE_INVENTORY_ITEM)).
		WithArgs(item_name, item_model,
			item_part_number, item_bar_code, item_has_bar_code, item_manual_code,
			t_item_id).
		WillReturnError(fmt.Errorf("update error"))

	tnx, _ := db.Begin()
	item := dummyTestItem()
	item.ItemId = t_item_id
	_, err := store.UpdateItemInTx(tnx, item)
	if err == nil {
		t.Errorf("expected error")
	}
}

func _itemQueryExpectation() *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_INVENTORY_ITEM))
}

func _itemQueryRows() sqlmock.Rows {
	return sqlmock.NewRows(_cols("item_id,company_id, "+
		"name, model_year, part_number,bar_code,has_bar_code,manual_code")).
		AddRow(t_item_id, t_company_id, item_name, item_model,
			item_part_number, item_bar_code, item_has_bar_code, item_manual_code)
}

func TestGetItemById(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	_itemQueryExpectation().
		WithArgs(t_item_id).
		WillReturnRows(_itemQueryRows())

	_, err := store.GetItemById(t_item_id)
	if err != nil {
		t.Errorf("GetItemById error %v", err)
	}
}

func TestGetItemByIdFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	_itemQueryExpectation().
		WithArgs(t_item_id).
		WillReturnError(fmt.Errorf("query error"))

	_, err := store.GetItemById(t_item_id)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGetCompanyItems(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	_itemQueryExpectation().
		WithArgs(t_company_id).
		WillReturnRows(_itemQueryRows())

	items, err := store.GetAllCompanyItems(t_company_id)
	if err != nil || len(items) != 1 {
		t.Errorf("GetItemById error %v", err)
	}
}

func TestGetCompanyItemsFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	_itemQueryExpectation().
		WithArgs(t_company_id).
		WillReturnError(fmt.Errorf("query error"))

	_, err := store.GetAllCompanyItems(t_company_id)
	if err == nil {
		t.Errorf("expected error")
	}
}
