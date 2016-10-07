package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

const (
	PERMISSION_JSON_MEMBER_ID         = "member_id"
	PERMISSION_JSON_MEMBER_PERMISSION = "member_permission"
)

// these are not exported
const (
	_PERMISSION_JSON_TYPE           = "permission_type"
	_PERMISSION_JSON_BRANCHES       = "branches"
	_PERMISSION_JSON_STORE_BRANCHES = "store_branches"
)

type PermType int64

const (
	// This is in a separate "const" block to reset "iota", otherwise it will be 4 here
	PERMISSION_TYPE_CREATOR PermType = iota + 1
	PERMISSION_TYPE_ADMIN
	PERMISSION_TYPE_MANAGER
	PERMISSION_TYPE_BRANCH_MANAGER
	PERMISSION_TYPE_BRANCH_CASHIER
	PERMISSION_TYPE_BRANCH_WORKER
)

type UserPermission struct {
	CompanyId int64
	UserId    int64

	// This is the form stored in data-store
	EncodedPermission string

	PermissionType PermType

	BranchesAllowed []int64
	StoresAllowed   []int64
}

type Pair_Company_UserPermission struct {
	CompanyInfo Company
	Permission  UserPermission
}

type Pair_User_UserPermission struct {
	Member     User
	Permission UserPermission
}

func (u *UserPermission) Encode() string {
	permission := map[string]interface{}{
		_PERMISSION_JSON_TYPE: u.PermissionType,
	}
	if u.BranchesAllowed != nil {
		permission[_PERMISSION_JSON_BRANCHES] = u.BranchesAllowed
	}
	if u.StoresAllowed != nil {
		permission[_PERMISSION_JSON_STORE_BRANCHES] = u.StoresAllowed
	}
	b, _ := json.Marshal(permission)
	u.EncodedPermission = string(b)
	return u.EncodedPermission
}

func DecodePermission(s string) (*UserPermission, error) {
	// TODO: yet to be implemented
	return nil, nil
}

func (b *shStore) SetUserPermission(p *UserPermission) (*UserPermission, error) {
	tnx, err := b.Begin()
	if err != nil {
		return nil, fmt.Errorf("Error setting permission '%v'", err)
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()

	permission, err := b.SetUserPermissionInTx(tnx, p)
	if err != nil {
		return nil, err
	}
	tnx.Commit()

	return permission, nil
}

func (b *shStore) SetUserPermissionInTx(tnx *sql.Tx, p *UserPermission) (*UserPermission, error) {
	rows, err := tnx.Query(
		fmt.Sprintf("select permission from %s "+
			"where company_id = $1 and user_id = $2", TABLE_U_PERMISSION),
		p.CompanyId, p.UserId)
	if err != nil {
		return nil, err
	}
	if rows.Next() { // if the user already had relations with company, update permission
		rows.Close()
		stmt := fmt.Sprintf("update %s set "+
			"permission = $1 "+
			"where company_id = $2 and user_id = $3", TABLE_U_PERMISSION)
		_, err = tnx.Exec(stmt, p.EncodedPermission, p.CompanyId, p.UserId)
		if err != nil {
			return nil, err
		}
	} else { // the user is new, add permission
		rows.Close()
		_, err = tnx.Exec(
			fmt.Sprintf("insert into %s "+
				"(company_id, user_id, permission) values "+
				"($1, $2, $3)", TABLE_U_PERMISSION),
			p.CompanyId, p.UserId, p.EncodedPermission)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (b *shStore) RemoveUserFromCompanyInTx(tnx *sql.Tx, user_id, company_id int64) error {
	_, err := tnx.Exec(fmt.Sprintf("delete from %s where user_id = $1 and company_id = $2", TABLE_U_PERMISSION),
		user_id, company_id)

	return err
}

func (b *shStore) GetUserPermission(u *User, company_id int64) (*UserPermission, error) {
	msg := fmt.Sprintf("error fetching user:'%d' permission for company:'%d'",
		u.UserId, company_id)
	permission, err := _queryUserPermission(b, msg,
		"where company_id = $1 and user_id = $2", company_id, u.UserId)
	if err != nil {
		return nil, err
	}
	return permission, nil
}

func (b *shStore) GetUserCompanyPermissions(u *User) ([]*Pair_Company_UserPermission, error) {
	var result []*Pair_Company_UserPermission
	query := fmt.Sprintf(
		"select c.company_id, c.company_name, c.encoded_payment, "+
			" p.company_id, p.user_id, p.permission "+
			"FROM %s AS c INNER JOIN %s AS p ON (c.company_id = p.company_id) "+
			"WHERE p.user_id = $1",
		TABLE_COMPANY, TABLE_U_PERMISSION)

	rows, err := b.Query(query, u.UserId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		pc := new(Pair_Company_UserPermission)
		err = rows.Scan(
			&pc.CompanyInfo.CompanyId,
			&pc.CompanyInfo.CompanyName,
			&pc.CompanyInfo.EncodedPayment,

			&pc.Permission.CompanyId,
			&pc.Permission.UserId,
			&pc.Permission.EncodedPermission,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrNoData
			}
			return nil, err
		}
		result = append(result, pc)
	}

	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}

func (b *shStore) GetCompanyMembersPermissions(c *Company) ([]*Pair_User_UserPermission, error) {
	var result []*Pair_User_UserPermission
	query := fmt.Sprintf(
		"select p.company_id, p.user_id, p.permission, "+
			" u.user_id, u.username "+
			" FROM %s AS p INNER JOIN %s AS u ON (p.user_id = u.user_id) "+
			" WHERE p.company_id = $1",
		TABLE_U_PERMISSION, TABLE_USER, TABLE_U_PERMISSION)

	rows, err := b.Query(query, c.CompanyId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		member_permission := new(Pair_User_UserPermission)
		err = rows.Scan(
			&member_permission.Permission.CompanyId,
			&member_permission.Permission.UserId,
			&member_permission.Permission.EncodedPermission,
			&member_permission.Member.UserId,
			&member_permission.Member.Username,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrNoData
			}
			return nil, err
		}
		result = append(result, member_permission)
	}

	if len(result) == 0 {
		return nil, ErrNoData
	}
	return result, nil
}

func _queryUserPermission(s *shStore, err_msg string, where_stmt string, args ...interface{}) (*UserPermission, error) {
	p := new(UserPermission)
	query := fmt.Sprintf(
		"select company_id, user_id, permission from %s", TABLE_U_PERMISSION)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = s.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = s.QueryRow(query)
	}

	if err := row.Scan(&p.CompanyId, &p.UserId, &p.EncodedPermission); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoData
		}
		return nil, fmt.Errorf("%s %s", err_msg, err)
	}

	return p, nil
}
