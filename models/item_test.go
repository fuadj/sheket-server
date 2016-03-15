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
	i := &ShItem{CompanyId: company_id,
		CategoryId: category_id, Name: item_name,
		ModelYear: item_model, PartNumber: item_part_number,
		BarCode: item_bar_code, HasBarCode: item_has_bar_code, ManualCode: item_manual_code}
	return i
}

func TestCreateInventoryItem(t *testing.T) {
	mock_setup(t, "TestCreateInventoryItem")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_INVENTORY_ITEM)).
		WithArgs(company_id, category_id, item_name, item_model,
		item_part_number, item_bar_code, item_has_bar_code, item_manual_code).
		WillReturnRows(sqlmock.NewRows(_cols("item_id")).AddRow(item_id))

	item, err := store.CreateItem(dummyTestItem())
	if err != nil {
		_log_err("CreateItem error '%v'", err)
	} else if item.ItemId != item_id {
		_log_err("Not the expected item")
	}
}

func TestCreateInventoryItemFail(t *testing.T) {
	mock_setup(t, "TestCreateInventoryItemFail")
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_INVENTORY_ITEM)).
		WithArgs(company_id, category_id, item_name, item_model,
		item_part_number, item_bar_code, item_has_bar_code, item_manual_code).
		WillReturnError(fmt.Errorf("insert error"))

	_, err := store.CreateItem(dummyTestItem())
	if err == nil {
		_log_err("expected error")
	}
}

func _itemQueryExpectation() *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_INVENTORY_ITEM))
}

func _itemQueryRows() sqlmock.Rows {
	return sqlmock.NewRows(_cols("item_id,company_id, category_id, "+
		"name, model_year, part_number,bar_code,has_bar_code,manual_code")).
		AddRow(item_id, company_id, category_id, item_name, item_model,
		item_part_number, item_bar_code, item_has_bar_code, item_manual_code)
}

func TestGetItemById(t *testing.T) {
	mock_setup(t, "TestGetItemById")
	defer mock_teardown()

	_itemQueryExpectation().
		WithArgs(item_id).
		WillReturnRows(_itemQueryRows())

	_, err := store.GetItemById(item_id)
	if err != nil {
		_log_err("GetItemById error %v", err)
	}
}

func TestGetItemByIdFail(t *testing.T) {
	mock_setup(t, "TestGetItemByIdFail")
	defer mock_teardown()

	_itemQueryExpectation().
		WithArgs(item_id).
		WillReturnError(fmt.Errorf("query error"))

	_, err := store.GetItemById(item_id)
	if err == nil {
		_log_err("expected error")
	}
}

func TestGetCompanyItems(t *testing.T) {
	mock_setup(t, "TestGetCompanyItems")
	defer mock_teardown()

	_itemQueryExpectation().
		WithArgs(company_id).
		WillReturnRows(_itemQueryRows())

	items, err := store.GetAllCompanyItems(company_id)
	if err != nil || len(items) != 1 {
		_log_err("GetItemById error %v", err)
	}
}

func TestGetCompanyItemsFail(t *testing.T) {
	mock_setup(t, "TestGetCompanyItemsFail")
	defer mock_teardown()

	_itemQueryExpectation().
	WithArgs(company_id).
	WillReturnError(fmt.Errorf("query error"))

	_, err := store.GetAllCompanyItems(company_id)
	if err == nil {
		_log_err("expected error")
	}
}
