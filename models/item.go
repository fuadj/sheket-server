package models

import (
	"database/sql"
	"fmt"
)

type ShItem struct {
	ItemId     int64	`json:"item_id"`
	CompanyId  int64	`json:"company_id"`
	CategoryId int64	`json:"category_id"`
	Name       string	`json:"name"`
	ModelYear  string	`json:"model_year"`
	PartNumber string	`json:"part_number"`
	BarCode    string	`json:"bar_code,omitempty"`
	HasBarCode bool		`json:"has_bar_code"`
	ManualCode string
}

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
			"(company_id, category_id, name, model_year, "+
			"part_number, bar_code, has_bar_code, manual_code) values "+
			"($1, $2, $3, $4, $5, $6, $7, $8) RETURNING item_id;", TABLE_INVENTORY_ITEM),
		item.CompanyId, item.CategoryId, item.Name, item.ModelYear,
		item.PartNumber, item.BarCode, item.HasBarCode, item.ManualCode).Scan(&item.ItemId)
	return item, err
}

func (s *shStore) UpdateItemInTx(tnx *sql.Tx, item *ShItem) (*ShItem, error) {
	_, err := tnx.Exec(
		fmt.Sprintf("update %s set "+
		"(category_id, name, model_year, "+
		"part_number, bar_code, has_bar_code, manual_code) values "+
		"($1, $2, $3, $4, $5, $6, $7) " +
		"where item_id = $8", TABLE_INVENTORY_ITEM),
		item.CategoryId, item.Name, item.ModelYear,
		item.PartNumber, item.BarCode, item.HasBarCode, item.ManualCode,
		item.ItemId)
	return item, err
}

func (s *shStore) GetItemById(id int64) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that id %d", id)
	item, err := _queryInventoryItems(s, msg, "where item_id = $1", id)
	if err != nil || len(item) == 0 {
		if err == nil {
			err = fmt.Errorf("No item with id:%d", id)
		}
		return nil, err
	}

	return item[0], nil
}

func (s *shStore) GetAllCompanyItems(company_id int64) ([]*ShItem, error) {
	msg := fmt.Sprintf("no item in company:%d", company_id)
	item, err := _queryInventoryItems(s, msg, "where company = $1", company_id)
	if err != nil || len(item) == 0 {
		if err == nil {
			err = fmt.Errorf("No items in company:%d", company_id)
		}
		return nil, err
	}

	return item, nil
}

func _queryInventoryItems(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShItem, error) {
	var result []*ShItem

	query := fmt.Sprintf("select item_id, company_id, category_id, name, model_year "+
		"part_number, bar_code, has_bar_code, manual_code from %s", TABLE_INVENTORY_ITEM)
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
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, i)
	}
	return result, nil
}
