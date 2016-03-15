package models
import (
	"fmt"
	"database/sql"
)

type ShItem struct {
	ItemId		int64
	CompanyId 		int64
	CategoryId		int64
	Name 			string
	ModelYear		string
	PartNumber		string
	BarCode 		string
	HasBarCode		bool
	ManualCode 		string
}

func (s *shStore) CreateItem(item *ShItem) (*ShItem, error) {
	err := s.QueryRow(
		fmt.Sprintf("insert into %s " +
			"(company_id, category_id, name, model_year, " +
			"part_number, bar_code, has_bar_code, manual_code) values " +
			"($1, $2, $3, $4, $5, $6, $7, $8) RETURNING item_id;", TABLE_INVENTORY_ITEM),
		item.CompanyId, item.CategoryId, item.Name, item.ModelYear,
		item.PartNumber, item.BarCode, item.HasBarCode, item.ManualCode).Scan(&item.ItemId)
	return item, err
}

func (s *shStore) GetItemById(id int64) (*ShItem, error) {
	msg := fmt.Sprintf("no item with that id %d", id)
	item, err := _queryInventoryItems(s, msg, "where item_id = $1", id)
	if err != nil || len(item) == 0 {
		if err == nil{
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
		if err == nil{
			err = fmt.Errorf("No items in company:%d", company_id)
		}
		return nil, err
	}

	return item, nil
}

func _queryInventoryItems(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShItem, error) {
	var result []*ShItem

	query := fmt.Sprintf("select item_id, company_id, category_id, name, model_year " +
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
