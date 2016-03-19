package models

import (
	"database/sql"
	"fmt"
)

type User struct {
	UserId         int64
	Username       string
	HashedPassword string
}

type UserPermission struct {
	CompanyId      int64
	UserId         int64
	PermissionType int64
	BranchId       int64
}

func (b *shStore) CreateUser(u *User) (*User, error) {
	tnx, err := b.Begin()
	if err != nil {
		return nil, fmt.Errorf("User create error '%v'", err)
	}
	defer func() {
		if err != nil {
			tnx.Rollback()
		}
	}()

	user, err := b.CreateUserInTx(tnx, u)
	if err != nil {
		// if user isn't nil, it means it already existed in the db
		return user, err
	}

	tnx.Commit()
	return user, nil
}

func (b *shStore) CreateUserInTx(tnx *sql.Tx, u *User) (*User, error) {
	prev_user, err := _queryUserTnx(tnx, "user already exists", "where username = $1", u.Username)
	if prev_user != nil {
		return prev_user, fmt.Errorf("User:'%s' already exists", u.Username)
	}

	err = tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(username, hashpass) values "+
			"($1, $2) returning user_id", TABLE_USER),
		u.Username, u.HashedPassword).Scan(&u.UserId)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (b *shStore) FindUserByName(username string) (*User, error) {
	msg := fmt.Sprintf("no user with name %s", username)
	user, err := _queryUser(b, msg, "where username = $1", username)
	if err != nil || user == nil {
		return nil, err
	}
	return user, nil
}

func (b *shStore) FindUserById(id int64) (*User, error) {
	msg := fmt.Sprintf("no user with id:%d", id)
	user, err := _queryUser(b, msg, "where user_id = $1", id)
	if err != nil || user == nil {
		return nil, err
	}
	return user, nil
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
		fmt.Sprintf("select permission_type from %s "+
			"where company_id = $1 and user_id = $2", TABLE_U_PERMISSION),
		p.CompanyId, p.UserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() { // if the user already had relations with company, update permission
		stmt := fmt.Sprintf("update %s set "+
			"permission_type = $1, branch_id = $2 "+
			"where company_id = $3 and user_id = $4", TABLE_U_PERMISSION)
		_, err = tnx.Exec(stmt, p.PermissionType, p.BranchId, p.CompanyId, p.UserId)
		if err != nil {
			return nil, err
		}
	} else { // the user is new, add permission
		_, err = tnx.Exec(
			fmt.Sprintf("insert into %s "+
				"(company_id, user_id, permission_type, branch_id) values "+
				"($1, $2, $3, $4)", TABLE_U_PERMISSION),
			p.CompanyId, p.UserId, p.PermissionType, p.BranchId)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (b *shStore) GetUserPermission(p *UserPermission) (*UserPermission, error) {
	msg := fmt.Sprintf("error fetching user:'%d' permission for company:'%d'",
		p.UserId, p.CompanyId)
	permission, err := _queryUserPermission(b, msg,
		"where company_id = $1 and user_id = $2", p.CompanyId, p.UserId)
	if err != nil || permission == nil {
		return nil, err
	}
	return permission, nil
}

func _queryUser(s *shStore, err_msg string, where_stmt string, args ...interface{}) (*User, error) {
	u := new(User)
	query := fmt.Sprintf("select id, username, hashpass from %s", TABLE_USER)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = s.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = s.QueryRow(query)
	}

	if err := row.Scan(&u.UserId, &u.Username, &u.HashedPassword); err != nil {
		return nil, fmt.Errorf("%s %s", err_msg, err)
	}

	return u, nil
}

func _queryUserTnx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) (*User, error) {
	u := new(User)
	query := fmt.Sprintf("select id, username, hashpass from %s", TABLE_USER)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = tnx.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = tnx.QueryRow(query)
	}

	if err := row.Scan(&u.UserId, &u.Username, &u.HashedPassword); err != nil {
		return nil, fmt.Errorf("%s %s", err_msg, err)
	}

	return u, nil
}

func _queryUserPermission(s *shStore, err_msg string, where_stmt string, args ...interface{}) (*UserPermission, error) {
	p := new(UserPermission)
	query := fmt.Sprintf(
		"select company_id, user_id, permission_type, branch_id from %s", TABLE_U_PERMISSION)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = s.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = s.QueryRow(query)
	}

	if err := row.Scan(&p.CompanyId, &p.UserId, &p.PermissionType, &p.PermissionType); err != nil {
		return nil, fmt.Errorf("%s %s", err_msg, err)
	}

	return p, nil
}
