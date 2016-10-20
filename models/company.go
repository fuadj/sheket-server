package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type Company struct {
	CompanyId      int
	CompanyName    string
	EncodedPayment string
}

const (
	PAYMENT_CONTRACT_NONE               = 0
	PAYMENT_CONTRACT_LIMITED_FREE       = 1
	PAYMENT_CONTRACT_UNLIMITED_ONE_TIME = 2
	PAYMENT_CONTRACT_SUBSCRIPTION       = 3
)

// can be used in either {employee | branch | item} to signal no limits
const PAYMENT_LIMIT_NONE = -1

type PaymentInfo struct {
	// value returned from time.Now().Unix(), time since the epoch. It is easier to store and "transport"
	IssuedDate int64

	ContractType   int
	DurationInDays int

	EmployeeLimit int
	BranchLimit   int
	ItemLimit     int
}

func (b *shStore) CreateCompanyInTx(tnx *sql.Tx, u *User, c *Company) (*Company, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(company_name, encoded_payment) values "+
			"($1, $2) returning company_id;", TABLE_COMPANY),
		c.CompanyName, c.EncodedPayment).Scan(&c.CompanyId)
	if err != nil {
		return nil, err
	}

	return c, err
}

func (b *shStore) UpdateCompanyInTx(tnx *sql.Tx, c *Company) (*Company, error) {
	_, err := tnx.Exec(
		fmt.Sprintf("update %s set "+
			" company_name = $1, encoded_payment = $2 "+
			" where company_id = $3 ", TABLE_COMPANY),
		c.CompanyName, c.EncodedPayment,
		c.CompanyId)
	return c, err
}

func (b *shStore) GetCompanyById(id int) (*Company, error) {
	msg := fmt.Sprintf("no company with id %d", id)
	companies, err := _queryCompany(b, msg, "where company_id = $1", id)
	if err != nil {
		return nil, err
	}
	return companies[0], nil
}

func _queryCompany(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*Company, error) {
	var result []*Company

	query := fmt.Sprintf("select company_id, company_name, encoded_payment from %s", TABLE_COMPANY)
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

	/**
	 * We need to use these Null* values and not directly scan into
	 * the attributes as that throws an error if the value is NULL.
	 *
	 * See: https://github.com/go-sql-driver/mysql/issues/34
	 */
	var _company_id sql.NullInt64
	var _name, _encoded_payment sql.NullString

	for rows.Next() {
		if err := rows.Scan(
			&_company_id,
			&_name,
			&_encoded_payment,
		); err == sql.ErrNoRows {
			// no-op
		} else if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		} else {
			c := new(Company)
			c.CompanyId = int(_company_id.Int64)
			c.CompanyName = _name.String
			c.EncodedPayment = _encoded_payment.String
			result = append(result, c)
		}
	}
	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}

const (
	_p_s_issued_date    = "issued_date"
	_p_s_duration       = "duration"
	_p_s_contract_type  = "contract_type"
	_p_s_employee_limit = "employee_limit"
	_p_s_branch_limit   = "branch_limit"
	_p_s_item_limit     = "item_limit"
)

const _C_D = ":%d"
const _C_D_S = _C_D + ";"

func (p *PaymentInfo) Encode() string {
	return fmt.Sprintf(
		_p_s_issued_date+_C_D_S+
			_p_s_duration+_C_D_S+
			_p_s_contract_type+_C_D_S+
			_p_s_employee_limit+_C_D_S+
			_p_s_branch_limit+_C_D_S+
			_p_s_item_limit+_C_D,

		p.IssuedDate, p.DurationInDays, p.ContractType,
		p.EmployeeLimit, p.BranchLimit, p.ItemLimit)
}

func DecodePayment(s string) (*PaymentInfo, error) {
	p := &PaymentInfo{}
	subs := strings.Split(s, ";")

	if len(subs) != 6 {
		return nil, fmt.Errorf("Invalid payment info encoding '%s'", s)
	}

	p.IssuedDate = int64(_extract_int(subs[0]))
	p.DurationInDays = _extract_int(subs[1])
	p.ContractType = _extract_int(subs[2], PAYMENT_CONTRACT_NONE)
	if p.ContractType == PAYMENT_CONTRACT_NONE {
		return nil, fmt.Errorf("invalid contract type '%d'", p.ContractType)
	}
	p.EmployeeLimit = _extract_int(subs[3])
	p.BranchLimit = _extract_int(subs[4])
	p.ItemLimit = _extract_int(subs[5])

	return p, nil
}

func _extract_int(s string, args ...int) int {
	var def int

	if len(args) != 0 {
		def = args[0]
	}

	subs := strings.Split(s, ":")
	if len(subs) != 2 {
		return def
	}

	i, err := strconv.Atoi(subs[1])
	if err != nil {
		return def
	}

	return i
}
