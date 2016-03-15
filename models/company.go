package models
import (
	"fmt"
	"database/sql"
"errors"
)

type Company struct {
	CompanyId 		int64
	CompanyName		string
	Contact 		string
}

func (b *shStore) CreateCompany(u *User, c *Company) (*Company, error) {
	tnx, err := b.Begin()
	if err != nil {
		return nil, fmt.Errorf("Company create error '%v'", err)
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()

	company, err := b.CreateCompanyInTransaction(tnx, u, c)
	if err != nil {
		return nil, err
	}

	tnx.Commit()

	return company, nil
}

func (b *shStore) CreateCompanyInTransaction(tnx *sql.Tx, u *User, c *Company) (*Company, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s " +
		"(company_name, contact) values " +
		"($1, $2) returning company_id;", TABLE_COMPANY),
		c.CompanyName, c.Contact).Scan(&c.CompanyId)
	if err != nil {
		return nil, err
	}

	return c, err
}

func (b *shStore) GetCompanyById(id int64) (*Company, error) {
	msg := fmt.Sprintf("no company with id %d", id)
	companies, err := _queryCompany(b, msg, "where company_id = $1", id)
	if err != nil {
		return nil, err
	}
	if len(companies) == 0 {
		return nil, errors.New(fmt.Sprintf("No company with id:%d", id))
	}
	return companies[0], nil
}

func _queryCompany(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*Company, error) {
	var result []*Company

	query := fmt.Sprintf("select company_id, company_name, contact from %s", TABLE_COMPANY)
	sort_by := " ORDER BY company_id desc"

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
		c := new(Company)
		err := rows.Scan(
			&c.CompanyId,
			&c.CompanyName,
			&c.Contact,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, c)
	}
	return result, nil
}