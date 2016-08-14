package models

import (
	"database/sql"
	"fmt"
)

const (
	AUTH_PROVIDER_FACEBOOK int64 = 1
	AUTH_PROVIDER_GOOGLE   int64 = 2
)

type User struct {
	// This is the id in our database
	UserId   int64
	Username string

	// these will be the id of the provider which gave us the user's info (like: facebook, ...)
	ProviderID     int64
	UserProviderID string
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

func (b *shStore) CreateUserInTx(tnx *sql.Tx, u *User) (*User, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(username, provider_id, user_provider_id) values "+
			"($1, $2, $3) returning user_id", TABLE_USER),
		u.Username, u.ProviderID, u.UserProviderID).
		Scan(&u.UserId)
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
	user, err := _queryUser(b,
		fmt.Sprintf("no user with id:%d", id),
		"where user_id = $1", id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (b *shStore) FindUserWithProviderIdInTx(tnx *sql.Tx, provider_id int64, provider_user_id string) (*User, error) {
	return _queryUserTnx(tnx,
		fmt.Sprintf("no user with id:%s in provider:%d", provider_user_id, provider_id),
		"where provider_id = $1 AND user_provider_id = $2",
		provider_id, provider_user_id)
}

func _queryUser(s *shStore, err_msg string, where_stmt string, args ...interface{}) (*User, error) {
	u := new(User)
	query := fmt.Sprintf("select user_id, username, provider_id, user_provider_id from %s", TABLE_USER)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = s.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = s.QueryRow(query)
	}

	err := row.Scan(&u.UserId, &u.Username, &u.ProviderID, &u.UserProviderID)
	return _checkUserError(u, err, err_msg)
}

func _queryUserTnx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) (*User, error) {
	u := new(User)
	query := fmt.Sprintf("select user_id, username, provider_id, user_provider_id from %s", TABLE_USER)

	var row *sql.Row
	if len(where_stmt) > 0 {
		row = tnx.QueryRow(query+" "+where_stmt, args...)
	} else {
		row = tnx.QueryRow(query)
	}

	err := row.Scan(&u.UserId, &u.Username, &u.ProviderID, &u.UserProviderID)
	return _checkUserError(u, err, err_msg)
}
