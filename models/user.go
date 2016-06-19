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

func _checkUserError(u *User, err error, err_msg string) (*User, error) {
	if err == nil {
		return u, nil
	} else if err == sql.ErrNoRows {
		return nil, ErrNoData
	} else {
		if err_msg == "" {
			return nil, err
		} else {
			return nil, fmt.Errorf("%s %s", err_msg, err.Error())
		}
	}
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
		return nil, err
	}

	tnx.Commit()
	return user, nil
}

func (b *shStore) CreateUserInTx(tnx *sql.Tx, u *User) (*User, error) {
	prev_user, err := _queryUserTnx(tnx, "query user error", "where username = $1", u.Username)
	if prev_user != nil {
		return nil, fmt.Errorf("username '%s' already exists", u.Username)
	}

	err = tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(username, hashpass) values "+
			"($1, $2) returning user_id", TABLE_USER),
		u.Username, u.HashedPassword).Scan(&u.UserId)
	return _checkUserError(u, err, "")
}

func (b *shStore) FindUserByName(username string) (*User, error) {
	msg := fmt.Sprintf("no user with name %s", username)
	user, err := _queryUser(b, msg, "where username = $1", username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (b *shStore) FindUserByNameInTx(tnx *sql.Tx, username string) (*User, error) {
	return _queryUserTnx(tnx, fmt.Sprintf("no user with %s name", username),
		"where username = $1", username)
}

func (b *shStore) FindUserById(id int64) (*User, error) {
	msg := fmt.Sprintf("no user with id:%d", id)
	user, err := _queryUser(b, msg, "where user_id = $1", id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func _queryUser(s *shStore, err_msg string, where_stmt string, args ...interface{}) (*User, error) {
	u := new(User)
	query := fmt.Sprintf("select user_id, username, hashpass from %s", TABLE_USER)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = s.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = s.QueryRow(query)
	}

	err := row.Scan(&u.UserId, &u.Username, &u.HashedPassword)
	return _checkUserError(u, err, err_msg)
}

func _queryUserTnx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) (*User, error) {
	u := new(User)
	query := fmt.Sprintf("select user_id, username, hashpass from %s", TABLE_USER)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = tnx.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = tnx.QueryRow(query)
	}

	err := row.Scan(&u.UserId, &u.Username, &u.HashedPassword)
	return _checkUserError(u, err, err_msg)
}
