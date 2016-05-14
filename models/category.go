package models

import (
	"database/sql"
	"fmt"
)

type ShCategory struct {
	CategoryId int64
	ClientUUID string
	CompanyId  int64
	ParentId   int64
	Name       string
}

const (
	CATEGORY_JSON_CATEGORY_ID = "category_id"
	CATEGORY_JSON_UUID        = "client_uuid"
	CATEGORY_JSON_COMPANY_ID  = "company_id"
	CATEGORY_JSON_PARENT_ID   = "parent_id"
	CATEGORY_JSON_NAME        = "name"
)

func (s *shStore) CreateCategory(category *ShCategory) (*ShCategory, error) {
	tnx, err := s.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()
	created, err := s.CreateCategoryInTx(tnx, category)
	if err != nil {
		return nil, err
	}
	tnx.Commit()
	return created, nil
}

func (s *shStore) CreateCategoryInTx(tnx *sql.Tx, category *ShCategory) (*ShCategory, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(company_id, name, parent_id, client_uuid) values "+
			"($1, $2, $3, $4) returning category_id;", TABLE_CATEGORY),
		category.CompanyId, category.Name, category.ParentId, category.ClientUUID).
		Scan(&category.CategoryId)
	return category, err
}

func (s *shStore) UpdateCategoryInTx(tnx *sql.Tx, category *ShCategory) (*ShCategory, error) {
	_, err := tnx.Exec(
		fmt.Sprintf("update %s set "+
			"name = $1, parent_id = $2, "+
			"where category_id = $3", TABLE_CATEGORY),
		category.Name, category.ParentId,
		category.CategoryId)
	return category, err
}

func (s *shStore) GetCategoryByUUIDInTx(tnx *sql.Tx, uid string) (*ShCategory, error) {
	msg := fmt.Sprintf("no category with that uuid:%s", uid)
	category, err := _queryCategoryInTx(tnx, msg, "where client_uuid = $1", uid)
	if err != nil {
		return nil, err
	}

	if len(category) == 0 {
		return nil, nil
	}
	return category[0], nil
}

func _queryCategoryInTx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) ([]*ShCategory, error) {
	var result []*ShCategory
	query := fmt.Sprintf("select category_id, company_id, name, parent_id, client_uuid from %s", TABLE_CATEGORY)
	sort_by := " ORDER BY category_id asc"

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
		c := new(ShCategory)
		err := rows.Scan(
			&c.CategoryId,
			&c.CompanyId,
			&c.Name,
			&c.ParentId,
			&c.ClientUUID,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %s", err_msg, err.Error())
		}

		result = append(result, c)
	}
	rows.Close()
	return result, nil
}