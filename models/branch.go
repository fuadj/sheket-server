package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type ShBranch struct {
	CompanyId int64
	BranchId  int64
	Name      string
	Location  string
}

func (b *ShBranch) Map() map[string]interface{} {
	result := make(map[string]interface{})
	result["company_id"] = b.CompanyId
	result["branch_id"] = b.BranchId
	result["name"] = b.Name
	result["location"] = b.Location
	return result
}

type ShBranchItem struct {
	CompanyId    int64
	BranchId     int64
	ItemId       int64
	Quantity     float64
	ItemLocation string
}

func (s *ShBranchItem) Map() map[string]interface{} {
	result := make(map[string]interface{})
	result["company_id"] = s.CompanyId
	result["branch_id"] = s.BranchId
	result["item_id"] = s.ItemId
	result["quantity"] = s.Quantity
	result["item_location"] = s.ItemLocation
	return result
}

func (s *shStore) CreateBranch(b *ShBranch) (*ShBranch, error) {
	tnx, err := s.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()
	created, err := s.CreateBranchInTx(tnx, b)
	if err != nil {
		return nil, err
	}
	tnx.Commit()
	return created, nil
}

func (s *shStore) CreateBranchInTx(tnx *sql.Tx, b *ShBranch) (*ShBranch, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
		"(company_id, branch_name, location) values "+
		"($1, $2, $3) returning branch_id;", TABLE_BRANCH),
		b.CompanyId, b.Name, b.Location).Scan(&b.BranchId)
	return b, err
}

func (s *shStore) UpdateBranchInTx(tnx *sql.Tx, b *ShBranch) (*ShBranch, error) {
	_, err := tnx.Exec(
		fmt.Sprintf("update %s set "+
		"(branch_name, location) values "+
		"($1, $2) where branch_id = $3 ", TABLE_BRANCH),
		b.Name, b.Location, b.BranchId)
	return b, err
}

func (s *shStore) GetBranchById(id int64) (*ShBranch, error) {
	msg := fmt.Sprintf("no branch with that id %d", id)
	branches, err := _queryBranch(s, msg, "where branch_id = $1", id)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return nil, errors.New(fmt.Sprintf("No branch with id:%d", id))
	}
	return branches[0], nil
}

func (s *shStore) ListCompanyBranches(id int64) ([]*ShBranch, error) {
	msg := fmt.Sprintf("error fetching branches of company:%d", id)
	branches, err := _queryBranch(s, msg, "where company_id = $1", id)
	if err != nil {
		return nil, err
	}
	return branches, nil
}

func (s *shStore) AddItemToBranch(item *ShBranchItem) (*ShBranchItem, error) {
	tnx, err := s.Begin()
	if err != nil {
		return nil, fmt.Errorf("Error creating item %v", err)
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()

	rows, err := tnx.Query(
		fmt.Sprintf("select item_id from %s "+
			"where branch_id = $1 and item_id = $2", TABLE_BRANCH_ITEM),
		item.BranchId, item.ItemId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() { // if the item already exists, overwrite it
		stmt := fmt.Sprintf("update %s set "+
			"quantity = $1, item_location = $2 "+
			"where branch_id = $3 and item_id = $4", TABLE_BRANCH_ITEM)
		_, err = tnx.Exec(stmt, item.Quantity, item.ItemLocation, item.BranchId, item.ItemId)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = tnx.Exec(
			fmt.Sprintf("insert into %s "+
				"(company_id, branch_id, item_id, quantity, item_location) values "+
				"($1, $2, $3, $4, $5)", TABLE_BRANCH_ITEM),
			item.CompanyId, item.BranchId, item.ItemId, item.Quantity, item.ItemLocation)
		if err != nil {
			return nil, err
		}
	}
	tnx.Commit()

	return item, nil
}
func (s *shStore) UpdateBranchItemInTx(tnx *sql.Tx, item *ShBranchItem) (*ShBranchItem, error) {
	_, err := tnx.Exec(fmt.Sprintf("update %s set "+
		"quantity = $1, item_location = $2 "+
		"where branch_id = $3 and item_id = $4", TABLE_BRANCH_ITEM),
		item.Quantity, item.ItemLocation, item.BranchId, item.ItemId)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *shStore) GetItemsInBranch(id int64) ([]*ShBranchItem, error) {
	msg := fmt.Sprintf("error fetching items in branch:%d", id)
	items, err := _queryBranchItem(s, msg, "where branch_id = $1", id)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *shStore) GetItemsInAllCompanyBranches(id int64) ([]*ShBranchItem, error) {
	msg := fmt.Sprintf("error fetching branches of company:%d", id)
	items, err := _queryBranchItem(s, msg, "where company_id = $1", id)
	if err != nil {
		return nil, err
	}
	return items, nil
}

/**
 * Below this are internal helper methods
 */
func _queryBranch(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShBranch, error) {
	var result []*ShBranch

	query := fmt.Sprintf("select company_id, branch_id, branch_name, location from %s", TABLE_BRANCH)
	sort_by := " ORDER BY branch_id desc"

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
		b := new(ShBranch)
		err := rows.Scan(
			&b.CompanyId,
			&b.BranchId,
			&b.Name,
			&b.Location,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, b)
	}
	return result, nil
}

func _queryBranchItem(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShBranchItem, error) {
	var result []*ShBranchItem

	query := fmt.Sprintf("select company_id, branch_id, item_id, quantity, item_location from %s",
		TABLE_BRANCH_ITEM)
	sort_by := " ORDER BY branch_id desc"

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
		b := new(ShBranchItem)
		err := rows.Scan(
			&b.CompanyId,
			&b.BranchId,
			&b.ItemId,
			&b.Quantity,
			&b.ItemLocation,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, b)
	}
	return result, nil
}
