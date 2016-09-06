package models

import (
	"database/sql"
	"fmt"
	"strings"
	"strconv"
)

type Company struct {
	CompanyId      int64
	CompanyName    string
	Contact        string
	EncodedPayment string
}

const (
	PAYMENT_JSON_COMPANY_ID = "company_id"
	PAYMENT_JSON_ISSUED_DATE = "issued_date"
	PAYMENT_JSON_CONTRACT_TYPE = "contract_type"
	PAYMENT_JSON_DURATION = "duration"
	PAYMENT_JSON_LIMIT_EMPLOYEE = "employee_limit"
	PAYMENT_JSON_LIMIT_BRANCH = "branch_limit"
	PAYMENT_JSON_LIMIT_ITEM = "item_limit"
)

const (
	PAYMENT_CONTRACT_TYPE_NONE int64 = 1
	PAYMENT_CONTRACT_TYPE_SINGLE_USE int64 = 2
	PAYMENT_CONTRACT_TYPE_FIRST_LEVEL int64 = 3
	PAYMENT_CONTRACT_TYPE_SECOND_LEVEL int64 = 4
)

// can be used in either {employee | branch | item} to signal no limits
const PAYMENT_LIMIT_NONE = -1

type PaymentInfo struct {
	// value returned from time.Now().Unix(), time since the epoch. It is easier to store and "transport"
	IssuedDate int64

	ContractType   int64
	DurationInDays int64

	EmployeeLimit int64
	BranchLimit   int64
	ItemLimit     int64
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

	company, err := b.CreateCompanyInTx(tnx, u, c)
	if err != nil {
		return nil, err
	}

	tnx.Commit()

	return company, nil
}

func (b *shStore) CreateCompanyInTx(tnx *sql.Tx, u *User, c *Company) (*Company, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(company_name, contact, encoded_payment) values "+
			"($1, $2, $3) returning company_id;", TABLE_COMPANY),
		c.CompanyName, c.Contact, c.EncodedPayment).Scan(&c.CompanyId)
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
	return companies[0], nil
}

func _queryCompany(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*Company, error) {
	var result []*Company

	query := fmt.Sprintf("select company_id, company_name, contact, encoded_payment from %s", TABLE_COMPANY)
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

	defer rows.Close()
	for rows.Next() {
		c := new(Company)
		if err := rows.Scan(
			&c.CompanyId,
			&c.CompanyName,
			&c.Contact,
			&c.EncodedPayment,
		); err == sql.ErrNoRows {
			// no-op
		} else if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		} else {
			result = append(result, c)
		}
	}
	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}
const (
	_p_s_issued_date = "issued_date"
	_p_s_duration = "duration"
	_p_s_contract_type = "contract_type"
	_p_s_employee_limit = "employee_limit"
	_p_s_branch_limit = "branch_limit"
	_p_s_item_limit = "item_limit"
)

const _C_D = ":%d"
const _C_D_S = _C_D + ";"

func (p *PaymentInfo) Encode() string {
	return fmt.Sprintf(
		_p_s_issued_date + _C_D_S +
		_p_s_duration + _C_D_S +
		_p_s_contract_type + _C_D_S +
		_p_s_employee_limit + _C_D_S +
		_p_s_branch_limit + _C_D_S +
		_p_s_item_limit + _C_D,

		p.IssuedDate, p.DurationInDays, p.ContractType,
		p.EmployeeLimit, p.BranchLimit, p.ItemLimit)
}

func DecodePayment(s string) (*PaymentInfo, error) {
	p := &PaymentInfo{}
	subs := strings.Split(s, ";")

	if len(subs) != 6 {
		return nil, fmt.Errorf("Invalid payment info encoding '%s'", s)
	}

	p.IssuedDate = _atoi(subs[0])
	p.DurationInDays = _atoi(subs[1])
	p.ContractType = _atoi(subs[2], PAYMENT_CONTRACT_TYPE_NONE)
	if p.ContractType == PAYMENT_CONTRACT_TYPE_NONE {
		return nil, fmt.Errorf("invalid contract type '%d'", p.ContractType)
	}
	p.EmployeeLimit = _atoi(subs[3])
	p.BranchLimit = _atoi(subs[4])
	p.ItemLimit = _atoi(subs[5])

	return p, nil
}

func _atoi(s string, def_val ...int64) int64 {
	i, err := strconv.Atoi(s)
	if err != nil && len(def_val) != 0 {
		return def_val[0]
	}

	return int64(i)
}

