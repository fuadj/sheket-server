package auth

import (
	"fmt"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"sheket/server/models"
	"github.com/gin-gonic/gin"
)

const (
	login_cookie    = "user_log_in"
	key_user_id     = "user_id"
	invalid_user_id = -1

	SESSION_COOKIE_LEN = 64

	loginRedirect = "redirect_url"

	// If user is prompted to login, a flash message will be saved
	// in a session that will redirect the user to the original page
	// they were session after login.
	REDIRECT_FLASH = "redirect_flash"
)

var (
	Store models.ShStore

	SessionStore = sessions.NewCookieStore([]byte(securecookie.GenerateRandomKey(SESSION_COOKIE_LEN)))

	// b/c the keys are randomly generated, the user will be logged out
	// if the server restarts, there by generating new keys and making the previous
	// cookie indecipherable.
	/*
	   cookieHandler = securecookie.New(
	   	securecookie.GenerateRandomKey(64),
	   	securecookie.GenerateRandomKey(32))
	*/
	cookieHandler = securecookie.New(
		GenerateDummyKey("abcd", 64),
		GenerateDummyKey("kkk", 32),
	)

	// This handler will be called when the user isn't logged-in
	// for handlers that require a logged-in user.
	// This is exposed for testing(mocking) purposes.
	NotLoggedInHandler = func(w http.ResponseWriter, r *http.Request) {
		// Send the user to the login page is the DEFAULT action.
		//http.Redirect(w, r, r_login, http.StatusFound)
	}
)

func RequireLogin(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsLoggedIn(c.Request) {
			_, err := GetCurrentUser(c.Request)
			if err != nil {
				fmt.Printf("Invalid login %v", err.Error())
			}
			c.String(http.StatusNonAuthoritativeInfo,
				fmt.Sprintf("%s requires a logged-in user", c.Request.URL.Path))
			return
		}
		h(c)
	}
}

func HashPassword(s string) string {
	var hashed []byte
	hashed, _ = bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	return string(hashed)
}

func CompareHashAndPassword(hashed string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
}

func GenerateDummyKey(s string, length int) []byte {
	k := make([]byte, length)

	for i := 0; i < length; i++ {
		k[i] = byte(s[i%len(s)])
	}
	return k
}

func IsLoggedIn(r *http.Request) bool {
	_, err := GetCurrentUser(r)
	return err == nil
}

// Checks if the user has required credentials
// Returns err if invalid
// If err is nil, the passed in {@link User} will
// have all its fields populated
func AuthenticateUser(u *models.User, password string) (*models.User, error) {
	var err error

	found, err := Store.FindUserByName(u.Username)
	if err != nil {
		return nil, err
	} else if found == nil {
		return nil, fmt.Errorf("invalid username")
	}

	if !CompareHashAndPassword(found.HashedPassword, password) {
		return nil, fmt.Errorf("invalid password")
	}

	return found, nil
}

// This gets the URL that was saved in a cookie when the user
// was forced to login from another page. It is the page url they were
// on initially on before they were redirected to the login page.
func GetRedirectURL(r *http.Request) string {
	var redirectURL string = ""

	session, err := SessionStore.Get(r, loginRedirect)
	if err == nil {
		if flashes := session.Flashes(REDIRECT_FLASH); len(flashes) > 0 {
			redirectURL = flashes[0].(string)
		}
	}

	return redirectURL
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
	var user_id int64 = invalid_user_id

	if cookie, err := r.Cookie(login_cookie); err == nil {
		decoded := make(map[string]int64)
		if err = cookieHandler.Decode(login_cookie, cookie.Value, &decoded); err == nil {
			user_id = decoded[key_user_id]
		} else {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("invalid login cookie")
	}

	return Store.FindUserById(user_id)
}
