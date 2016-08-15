package auth

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"net/http"
	"sheket/server/models"
)

const (
	login_cookie    = "user_log_in"
	key_user_id     = "user_id"
	invalid_user_id = -1

	SESSION_COOKIE_LEN = 64
)

var (
	Store models.ShStore

	SessionStore = sessions.NewCookieStore([]byte(securecookie.GenerateRandomKey(SESSION_COOKIE_LEN)))

	cookieHandler = securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
)

func IsUserLoggedIn(r *http.Request) bool {
	_, err := GetCurrentUserId(r)
	return err == nil
}

func RequireLogin(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsUserLoggedIn(c.Request) {
			c.String(http.StatusUnauthorized,
				fmt.Sprintf("%s requires a logged-in user", c.Request.URL.Path))
			return
		}
		h(c)
	}
}

func LoginUser(u *models.User, w http.ResponseWriter) {
	value := map[string]int64{
		key_user_id: u.UserId,
	}
	if encoded, err := cookieHandler.Encode(login_cookie, value); err == nil {
		cookie := &http.Cookie{
			Name:  login_cookie,
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}
}

func LogoutUser(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   login_cookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

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
