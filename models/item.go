package models

import (
	"database/sql"
	"fmt"
	"strings"
)

type ShItem struct {
	ItemId     int64
	ClientUUID string
	CompanyId  int64
	CategoryId int64
	Name       string
	ItemCode   string

	UnitOfMeasurement int64
	HasDerivedUnit    bool
	DerivedName       string
	DerivedFactor     float64
	ReorderLevel      float64

	ModelYear  string
	PartNumber string
	BarCode    string
	HasBarCode bool

	StatusFlag int64
}

const (
	_db_item_id          = " item_id "
	_db_item_client_uuid = " client_uuid "
	_db_item_company_id  = " company_id "
	_db_item_category_id = " category_id "
	_db_item_name        = " item_name "
	_db_item_code        = " item_code "

	_db_item_units            = " units "
	_db_item_has_derived_unit = " has_derived_unit "
	_db_item_derived_name     = " derived_name "
	_db_item_derived_factor   = " derived_factor "
	_db_item_reorder_level    = " reorder_level "

	_db_item_model_year   = " model_year "
	_db_item_part_number  = " part_number "
	_db_item_bar_code     = " bar_code "
	_db_item_has_bar_code = " has_bar_code "
)

const (
	// Shared across entities that can have this field. Currently used to track if is
	// visible/invisible. This is an integer, default value is STATUS_VISIBLE.
	_db_status_flag  = "status_flag"
	JSON_STATUS_FLAG = "status_flag"

	STATUS_VISIBLE    int64 = 1
	STATUS_IN_VISIBLE int64 = 2
)

const (
	ITEM_JSON_ITEM_ID     = "item_id"
	ITEM_JSON_UUID        = "client_uuid"
	ITEM_JSON_COMPANY_ID  = "company_id"
	ITEM_JSON_CATEGORY_ID = "category_id"
	ITEM_JSON_ITEM_NAME   = "item_name"
	ITEM_JSON_ITEM_CODE   = "item_code"

	ITEM_JSON_UNIT_OF_MEASUREMENT = "units"
	ITEM_JSON_HAS_DERIVED_UNIT    = "has_derived_unit"
	ITEM_JSON_DERIVED_NAME        = "derived_name"
	ITEM_JSON_DERIVED_FACTOR      = "derived_factor"
	ITEM_JSON_REORDER_LEVEL       = "reorder_level"

	ITEM_JSON_MODEL_YEAR   = "model_year"
	ITEM_JSON_PART_NUMBER  = "part_number"
	ITEM_JSON_BAR_CODE     = "bar_code"
	ITEM_JSON_HAS_BAR_CODE = "has_bar_code"
)

func _checkItemArrError(items []*ShItem, err error) ([]*ShItem, error) {
	if err == nil {
		if len(items) == 0 {
			return nil, ErrNoData
		}
		return items, nil
	} else if err == sql.ErrNoRows {
		return nil, ErrNoData
	} else if err == ErrNoData {
		return nil, err
	} else {
		return nil, err
	}
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
		"insert into "+TABLE_INVENTORY_ITEM+" ( "+
			_db_item_client_uuid+", "+
			_db_item_company_id+", "+
			_db_item_category_id+", "+
			_db_item_name+", "+
			_db_item_code+", "+

			_db_item_units+", "+
			_db_item_has_derived_unit+", "+
			_db_item_derived_name+", "+
			_db_item_derived_factor+", "+
			_db_item_reorder_level+", "+

			_db_item_model_year+", "+
			_db_item_part_number+", "+
			_db_item_bar_code+", "+
			_db_item_has_bar_code+", "+
			_db_status_flag+") VALUES "+
			"($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) "+
			"returning "+_db_item_id+";",
		item.ClientUUID, item.CompanyId, item.CategoryId, item.Name, item.ItemCode,
		item.UnitOfMeasurement, item.HasDerivedUnit, item.DerivedName, item.DerivedFactor, item.ReorderLevel,
		item.ModelYear, item.PartNumber, item.BarCode, item.HasBarCode, item.StatusFlag).
		Scan(&item.ItemId)
	return item, err
}

func (s *shStore) UpdateItemInTx(tnx *sql.Tx, item *ShItem) (*ShItem, error) {
	_, err := tnx.Exec(
		"update "+TABLE_INVENTORY_ITEM+" set "+
			_db_item_name+" = $1, "+
			_db_item_code+" = $2, "+
			_db_item_category_id+" = $3, "+

			_db_item_units+" = $4, "+
			_db_item_has_derived_unit+" = $5, "+
			_db_item_derived_name+" = $6, "+
			_db_item_derived_factor+" = $7, "+
			_db_item_reorder_level+" = $8, "+

			_db_item_model_year+" = $9, "+
			_db_item_part_number+" = $10, "+
			_db_item_bar_code+" = $11, "+
			_db_item_has_bar_code+" = $12, "+
			_db_status_flag+" = $13 "+
			" where "+_db_item_id+" = $14",
		item.Name, item.ItemCode, item.CategoryId,
		item.UnitOfMeasurement, item.HasDerivedUnit, item.DerivedName, item.DerivedFactor, item.ReorderLevel,
		item.ModelYear, item.PartNumber, item.BarCode, item.HasBarCode, item.StatusFlag,
		item.ItemId)
	return item, err
}

func (s *shStore) GetItemByUUIDInTx(tnx *sql.Tx, uid string) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that uuid:%s", uid)
	items, err := _queryInventoryItemsInTx(tnx, msg, "where client_uuid = $1", uid)

	items, err = _checkItemArrError(items, err)
	if err != nil {
		return nil, err
	}

	return items[0], nil
}

func (s *shStore) GetItemById(id int64) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that id %d", id)
	items, err := _queryInventoryItems(s, msg, "where item_id = $1", id)

	items, err = _checkItemArrError(items, err)
	if err != nil {
		return nil, err
	}

	return items[0], nil
}

func (s *shStore) GetItemByIdInTx(tnx *sql.Tx, id int64) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that id %d", id)
	items, err := _queryInventoryItemsInTx(tnx, msg, "where item_id = $1", id)

	items, err = _checkItemArrError(items, err)
	if err != nil {
		return nil, err
	}

	return items[0], nil
}

func _queryInventoryItems(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShItem, error) {
	var result []*ShItem

	query := "select " +
		strings.Join(
			[]string{
				_db_item_id, _db_item_client_uuid, _db_item_company_id, _db_item_category_id,
				_db_item_name, _db_item_code,
				_db_item_units, _db_item_has_derived_unit, _db_item_derived_name, _db_item_derived_factor,
				_db_item_reorder_level,
				_db_item_model_year, _db_item_part_number, _db_item_bar_code, _db_item_has_bar_code,
				_db_status_flag,
			},
			", ") +
		"from " + TABLE_INVENTORY_ITEM
	sort_by := " ORDER BY " + _db_item_id + " desc"

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

	defer rows.Close()

	for rows.Next() {
		i := new(ShItem)
		err := rows.Scan(
			&i.ItemId,
			&i.ClientUUID,
			&i.CompanyId,
			&i.CategoryId,
			&i.Name,
			&i.ItemCode,
			&i.UnitOfMeasurement,
			&i.HasDerivedUnit,
			&i.DerivedName,
			&i.DerivedFactor,
			&i.ReorderLevel,
			&i.ModelYear,
			&i.PartNumber,
			&i.BarCode,
			&i.HasBarCode,
			&i.StatusFlag,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrNoData
			}
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, i)
	}

	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}

func _queryInventoryItemsInTx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) ([]*ShItem, error) {
	var result []*ShItem

	query := "select " +
		strings.Join(
			[]string{
				_db_item_id, _db_item_client_uuid, _db_item_company_id, _db_item_category_id,
				_db_item_name, _db_item_code,
				_db_item_units, _db_item_has_derived_unit, _db_item_derived_name, _db_item_derived_factor,
				_db_item_reorder_level,
				_db_item_model_year, _db_item_part_number, _db_item_bar_code, _db_item_has_bar_code,
				_db_status_flag,
			},
			", ") +
		"from " + TABLE_INVENTORY_ITEM
	sort_by := " ORDER BY " + _db_item_id + " desc"

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

	defer rows.Close()

	for rows.Next() {
		i := new(ShItem)
		err := rows.Scan(
			&i.ItemId,
			&i.ClientUUID,
			&i.CompanyId,
			&i.CategoryId,
			&i.Name,
			&i.ItemCode,
			&i.UnitOfMeasurement,
			&i.HasDerivedUnit,
			&i.DerivedName,
			&i.DerivedFactor,
			&i.ReorderLevel,
			&i.ModelYear,
			&i.PartNumber,
			&i.BarCode,
			&i.HasBarCode,
			&i.StatusFlag,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrNoData
			}
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, i)
	}

	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}
