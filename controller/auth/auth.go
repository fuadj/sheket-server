package auth

import (
	"fmt"
	"github.com/gorilla/securecookie"
	"net/http"
	"sheket/server/models"
)

const (
	login_cookie    = "user_log_in"
	key_user_id     = "user_id"
	invalid_user_id = -1
)

var (
	Store models.ShStore

	cookieHandler = securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
)

func GetCurrentUser(r *http.Request) (*models.User, error) {
	if user_id, err := GetCurrentUserId(r); err == nil {
		return Store.FindUserById(user_id)
	}
	return nil, fmt.Errorf("Can't find User")
}

func GetCurrentUserId(r *http.Request) (int64, error) {
	if cookie, err := r.Cookie(login_cookie); err == nil {
		decoded := make(map[string]int64)
		if err = cookieHandler.Decode(login_cookie, cookie.Value, &decoded); err == nil {
			return decoded[key_user_id], nil
		} else {
			return invalid_user_id, err
		}
	} else {
		return invalid_user_id, fmt.Errorf("invalid login cookie")
	}
}

func GetUser(login_cookie string) (*models.User, error) {
	if user_id, err := GetUserId(login_cookie); err == nil {
		return Store.FindUserById(user_id)
	} else {
		return nil, err
	}
}

func GetUserId(cookie string) (int64, error) {
	decoded := make(map[string]int64)
	if err := cookieHandler.Decode(login_cookie, cookie, &decoded); err == nil {
		return decoded[key_user_id], nil
	} else {
		return invalid_user_id, err
	}
}

func GenerateLoginCookie(u *models.User) (string, error) {
	var cookie string
	var err error

	if cookie, err = cookieHandler.Encode(
		login_cookie,
		map[string]int64{
			key_user_id: u.UserId,
		}); err == nil {
		return cookie, nil
	}
	return "", err
}
