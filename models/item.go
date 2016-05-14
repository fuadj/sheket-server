package models

import (
	"database/sql"
	"fmt"
)

type ShItem struct {
	ItemId     int64
	ClientUUID string
	CompanyId  int64
	CategoryId int64
	Name       string
	ModelYear  string
	PartNumber string
	BarCode    string
	HasBarCode bool
	ManualCode string
}

const (
	ITEM_JSON_ITEM_ID      = "item_id"
	ITEM_JSON_UUID         = "client_uuid"
	ITEM_JSON_COMPANY_ID   = "company_id"
	ITEM_JSON_CATEGORY_ID  = "category_id"
	ITEM_JSON_ITEM_NAME    = "item_name"
	ITEM_JSON_MODEL_YEAR   = "model_year"
	ITEM_JSON_PART_NUMBER  = "part_number"
	ITEM_JSON_BAR_CODE     = "bar_code"
	ITEM_JSON_HAS_BAR_CODE = "has_bar_code"
	ITEM_JSON_MANUAL_CODE  = "manual_code"
)

func (s *shStore) CreateItem(item *ShItem) (*ShItem, error) {
	tnx, err := s.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()
	created_item, err := s.CreateItemInTx(tnx, item)
	if err != nil {
		return nil, err
	}
	tnx.Commit()
	return created_item, nil
}

func (s *shStore) CreateItemInTx(tnx *sql.Tx, item *ShItem) (*ShItem, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(company_id, name, model_year, "+
			"part_number, bar_code, has_bar_code, manual_code, client_uuid, category_id) values "+
			"($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING item_id;", TABLE_INVENTORY_ITEM),
		item.CompanyId, item.Name, item.ModelYear,
		item.PartNumber, item.BarCode, item.HasBarCode, item.ManualCode, item.ClientUUID, item.CategoryId).
		Scan(&item.ItemId)
	return item, err
}

func (s *shStore) UpdateItemInTx(tnx *sql.Tx, item *ShItem) (*ShItem, error) {
	_, err := tnx.Exec(
		fmt.Sprintf("update %s set "+
			"name = $1, model_year = $2, "+
			" part_number = $3, bar_code = $4, has_bar_code = $5, manual_code = $6, category_id = $7 "+
			" where item_id = $8", TABLE_INVENTORY_ITEM),
		item.Name, item.ModelYear, item.PartNumber,
		item.BarCode, item.HasBarCode, item.ManualCode, item.CategoryId,
		item.ItemId)
	return item, err
}

func (s *shStore) GetItemByUUIDInTx(tnx *sql.Tx, uid string) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that uuid:%s", uid)
	item, err := _queryInventoryItemsInTx(tnx, msg, "where client_uuid = $1", uid)
	if err != nil {
		return nil, err
	}

	if len(item) == 0 {
		return nil, nil
	}

	return item[0], nil
}

func (s *shStore) GetItemById(id int64) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that id %d", id)
	item, err := _queryInventoryItems(s, msg, "where item_id = $1", id)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, fmt.Errorf("error getting item:%d", id)
	}

	return item[0], nil
}

func (s *shStore) GetItemByIdInTx(tnx *sql.Tx, id int64) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that id %d", id)
	item, err := _queryInventoryItemsInTx(tnx, msg, "where item_id = $1", id)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, fmt.Errorf("error getting item:%d", id)
	}

	return item[0], nil
}

func (s *shStore) GetAllCompanyItems(company_id int64) ([]*ShItem, error) {
	msg := fmt.Sprintf("no item in company:%d", company_id)
	item, err := _queryInventoryItems(s, msg, "where company = $1", company_id)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, fmt.Errorf("error getting items in company:%d", company_id)
	}

	return item, nil
}

func _queryInventoryItems(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShItem, error) {
	var result []*ShItem

	query := fmt.Sprintf("select item_id, company_id, category_id, name, model_year, "+
		"part_number, bar_code, has_bar_code, manual_code, client_uuid from %s", TABLE_INVENTORY_ITEM)
	sort_by := " ORDER BY item_id desc"

	var rows *sql.Rows
	var err error
	if len(where_stmt) > 0 {
		rows, err = s.Query(query+" "+where_stmt+sort_by, args...)
	} else {
		rows, err = s.Query(query + sort_by)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v", err_msg, err)
	}

	for rows.Next() {
		i := new(ShItem)
		err := rows.Scan(
			&i.ItemId,
			&i.CompanyId,
			&i.CategoryId,
			&i.Name,
			&i.ModelYear,
			&i.PartNumber,
			&i.BarCode,
			&i.HasBarCode,
			&i.ManualCode,
			&i.ClientUUID,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, i)
	}
	return result, nil
}

func _queryInventoryItemsInTx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) ([]*ShItem, error) {
	var result []*ShItem

	query := fmt.Sprintf("select item_id, company_id, category_id, name, model_year, "+
		"part_number, bar_code, has_bar_code, manual_code, client_uuid from %s", TABLE_INVENTORY_ITEM)
	sort_by := " ORDER BY item_id desc"

	var rows *sql.Rows
	var err error
	if len(where_stmt) > 0 {
		rows, err = tnx.Query(query+" "+where_stmt+sort_by, args...)
	} else {
		rows, err = tnx.Query(query + sort_by)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v", err_msg, err)
	}

	for rows.Next() {
		i := new(ShItem)
		err := rows.Scan(
			&i.ItemId,
			&i.CompanyId,
			&i.CategoryId,
			&i.Name,
			&i.ModelYear,
			&i.PartNumber,
			&i.BarCode,
			&i.HasBarCode,
			&i.ManualCode,
			&i.ClientUUID,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, i)
	}
	rows.Close()
	return result, nil
}
