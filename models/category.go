package models

import (
	"database/sql"
	"fmt"
)

const (
	CATEGORY_JSON_CATEGORY_ID = "category_id"
	CATEGORY_JSON_UUID        = "client_uuid"
	CATEGORY_JSON_PARENT_ID   = "parent_id"
	CATEGORY_JSON_NAME        = "name"
)

const (
	// branch category constants
	BRANCH_CATEGORY_JSON_ID = "branch_category_id"
	// It has no other fields
)

const (
	ROOT_CATEGORY_ID   = 1
	ROOT_CATEGORY_NAME = "__root category__"
)

type ShCategory struct {
	CategoryId int64
	ClientUUID string
	CompanyId  int64
	ParentId   int64
	Name       string
}

type ShBranchCategory struct {
	CompanyId 		int64
	BranchId 		int64
	CategoryId 		int64
}

func runInTransaction(s *shStore, f func(*sql.Tx) (*ShCategory, error)) (*ShCategory, error) {
	tnx, err := s.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()
	result, err := f(tnx)
	if err != nil {
		return nil, err
	}
	tnx.Commit()
	return result, nil
}

func (s *shStore) CreateCategory(category *ShCategory) (*ShCategory, error) {
	return runInTransaction(s,
		func(tnx *sql.Tx) (*ShCategory, error) {
			return s.CreateCategoryInTx(tnx, category)
		})
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
			"name = $1, parent_id = $2 "+
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
	return category[0], nil
}

func (s *shStore) GetCategoryById(id int64) (*ShCategory, error) {
	return runInTransaction(s,
		func(tnx *sql.Tx) (*ShCategory, error) {
			return s.GetCategoryByIdInTx(tnx, id)
		})
}

func (s *shStore) GetCategoryByIdInTx(tnx *sql.Tx, id int64) (*ShCategory, error) {
	msg := fmt.Sprintf("no category with that id:%d", id)
	category, err := _queryCategoryInTx(tnx, msg, "where category_id = $1", id)
	if err != nil {
		return nil, err
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

	defer rows.Close()

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
			if err == sql.ErrNoRows {
				return nil, ErrNoData
			}
			return nil, fmt.Errorf("%s %s", err_msg, err.Error())
		}

		result = append(result, c)
	}

	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}

func (s *shStore) AddCategoryToBranch(tnx *sql.Tx, branch_category *ShBranchCategory) (*ShBranchCategory, error) {
	rows, err := tnx.Query(
		fmt.Sprintf("select branch_id from %s " +
			"where branch_id = $1 and category_id = $2", TABLE_BRANCH_CATEGORY),
		branch_category.BranchId, branch_category.CategoryId)
	if err != nil {
		return nil, err
	}
	if rows.Next() {		// if the category exists inside the branch, we're done
		rows.Close()
	} else {
		rows.Close()
		_, err = tnx.Exec(
			fmt.Sprintf("insert into %s " +
				"(company_id, branch_id, category_id) values " +
				"($1, $2, $3)", TABLE_BRANCH_CATEGORY),
			branch_category.CompanyId, branch_category.BranchId, branch_category.CategoryId)
		if err != nil {
			return nil, err
		}
	}

	return branch_category, nil
}

func (s *shStore) GetBranchCategory(branch_id, category_id int64) (*ShBranchCategory, error) {
	err_msg := fmt.Sprintf("err fetching category:%d in branch:%d", category_id, branch_id)
	categories, err := _queryBranchCategory(s, err_msg, "where branch_id = $1 and category_id = $2",
		branch_id, category_id)
	if err != nil {
		return nil, err
	}
	return categories[0], nil
}

func (s *shStore) GetBranchCategoryInTx(tnx *sql.Tx, branch_id, category_id int64) (*ShBranchCategory, error) {
	err_msg := fmt.Sprintf("err fetching category:%d in branch:%d", category_id, branch_id)
	categories, err := _queryBranchCategoryInTx(s, err_msg, "where branch_id = $1 and category_id = $2",
		branch_id, category_id)
	if err != nil {
		return nil, err
	}
	return categories[0], nil
}

func _queryBranchCategory(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShBranchCategory, error) {
	var result []*ShBranchCategory

	query := fmt.Sprintf("select company_id, branch_id, category_id from %s",
		TABLE_BRANCH_CATEGORY)
	sort_by := " ORDER BY branch_id desc"

	var rows *sql.Rows
	var err error
	if len(where_stmt) > 0 {
		rows, err = s.Query(query + " " + where_stmt + sort_by, args...)
	} else {
		rows, err = s.Query(query + sort_by)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v", err_msg, err)
	}

	defer rows.Close()

	for rows.Next() {
		bc := new(ShBranchCategory)
		if err := rows.Scan(
			&bc.CompanyId,
			&bc.BranchId,
			&bc.CategoryId,
		); err == sql.ErrNoRows {
			// no-op
		} else if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		} else {
			result = append(result, bc)
		}
	}

	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}

func _queryBranchCategoryInTx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) ([]*ShBranchCategory, error) {
	var result []*ShBranchCategory

	query := fmt.Sprintf("select company_id, branch_id, category_id from %s",
		TABLE_BRANCH_CATEGORY)
	sort_by := " ORDER BY branch_id desc"

	var rows *sql.Rows
	var err error
	if len(where_stmt) > 0 {
		rows, err = tnx.Query(query + " " + where_stmt + sort_by, args...)
	} else {
		rows, err = tnx.Query(query + sort_by)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v", err_msg, err)
	}

	defer rows.Close()

	for rows.Next() {
		bc := new(ShBranchCategory)
		if err := rows.Scan(
			&bc.CompanyId,
			&bc.BranchId,
			&bc.CategoryId,
		); err == sql.ErrNoRows {
			// no-op
		} else if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		} else {
			result = append(result, bc)
		}
	}

	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}
