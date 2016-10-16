package auth

import (
	"github.com/gorilla/securecookie"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"sheket/server/models"
)

const (
	name_login_cookie = "user_log_in"
	key_user_id     = "user_id"
	invalid_user_id = -1
)

var (
	Store models.ShStore

	cookieHandler = securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
)

func GetUser(login_cookie string) (*models.User, error) {
	if user_id, err := GetUserId(login_cookie); err == nil {
		return Store.FindUserById(user_id)
	} else {
		return nil, grpc.Errorf(codes.Unauthenticated, "Invalid login")
	}
}

func GetUserId(cookie string) (int, error) {
	decoded := make(map[string]int)
	if err := cookieHandler.Decode(name_login_cookie, cookie, &decoded); err == nil {
		return decoded[key_user_id], nil
	} else {
		return invalid_user_id, err
	}
}

func GenerateLoginCookie(u *models.User) (string, error) {
	var cookie string
	var err error

	if cookie, err = cookieHandler.Encode(
		name_login_cookie,
		map[string]int{
			key_user_id: u.UserId,
		}); err == nil {
		return cookie, nil
	}
	return "", err
}
