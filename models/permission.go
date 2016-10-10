package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"strings"
)

const (
	PERMISSION_JSON_TYPE     = "permission_type"
	PERMISSION_JSON_BRANCHES = "branches"
)

const (
	PERMISSION_TYPE_OWNER    = 1
	PERMISSION_TYPE_MANAGER  = 2
	PERMISSION_TYPE_EMPLOYEE = 3
)

type BranchAuthority struct {
	BranchId  int `json:"branch_id"`
	Authority int `json:"authority"`
}

// i couldn't embed these constants in BranchAuthority json annotation,
// so they need to be kept up-to-date with it.
const (
	_json_branch_id = "branch_id"
	_json_authority = "authority"
)

type UserPermission struct {
	CompanyId int
	UserId    int

	PermissionType int

	Branches []BranchAuthority

	// This is used when retrieving UserPermission objects.
	// The permission will be stored in this.
	EncodedPermission string
}

type Pair_Company_UserPermission struct {
	CompanyInfo Company
	Permission  UserPermission
}

type Pair_User_UserPermission struct {
	Member     User
	Permission UserPermission
}

func (u *UserPermission) HasManagerAccess() bool {
	switch (u.PermissionType) {
	case PERMISSION_TYPE_OWNER, PERMISSION_TYPE_MANAGER:
		return true
	default:
		return false
	}
}

func (u *UserPermission) Encode() string {
	permission := map[string]interface{}{
		PERMISSION_JSON_TYPE: u.PermissionType,
	}
	if u.Branches != nil {
		permission[PERMISSION_JSON_BRANCHES] = u.Branches
	}
	b, _ := json.Marshal(permission)
	u.EncodedPermission = string(b)
	return u.EncodedPermission
}

func get_int(key string, field map[string]interface{}) (int, bool) {
	if val, ok := field[key]; ok {
		number, ok := val.(json.Number)
		if !ok {
			return -1, false
		}
		int_val, err := number.Int64()
		if err != nil {
			return -1, false
		}
		return int(int_val), true
	}
	return -1, false
}

// decodes the permission stored in UserPermission.Permission an populates the fields.
func (u *UserPermission) Decode() error {
	data, err := simplejson.NewFromReader(strings.NewReader(u.EncodedPermission))
	if err != nil {
		return err
	}
	if u.PermissionType, err = data.Get(PERMISSION_JSON_TYPE).Int(); err != nil {
		return err
	}
	if branches, ok := data.CheckGet(PERMISSION_JSON_BRANCHES); ok {
		arr, err := branches.Array()
		if err != nil {
			return err
		}
		for i := 0; i < len(arr); i++ {
			branch_authority, ok := arr[i].(map[string]interface{})
			if !ok {
				return fmt.Errorf("Error parsing branch authority at %d, '%v'", i+1, arr[i])
			}
			branch_id, ok := get_int(_json_branch_id, branch_authority)
			if !ok {
				return fmt.Errorf("Error parsing branchid at '%v'", arr[i])
			}
			authority, ok := get_int(_json_authority, branch_authority)
			if !ok {
				return fmt.Errorf("Error parsing branchid at '%v'", arr[i])
			}
			u.Branches = append(u.Branches,
				BranchAuthority{
					BranchId:  branch_id,
					Authority: authority,
				})
		}
	}

	return nil
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
	encoded := p.Encode()

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
		_, err = tnx.Exec(stmt, encoded, p.CompanyId, p.UserId)
		if err != nil {
			return nil, err
		}
	} else { // the user is new, add permission
		rows.Close()
		_, err = tnx.Exec(
			fmt.Sprintf("insert into %s "+
				"(company_id, user_id, permission) values "+
				"($1, $2, $3)", TABLE_U_PERMISSION),
			p.CompanyId, p.UserId, encoded)
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
			" FROM %s AS c INNER JOIN %s AS p ON (c.company_id = p.company_id) "+
			" WHERE p.user_id = $1",
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

		if err = pc.Permission.Decode(); err != nil {
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

		if err = member_permission.Permission.Decode(); err != nil {
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

	if err := p.Decode(); err != nil {
		return nil, err
	}

	return p, nil
}
