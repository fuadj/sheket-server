package controller

import (
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	fb "github.com/huandu/facebook"
	"net/http"
	"sheket/server/controller/auth"
	"sheket/server/models"
	"strings"
)

const (
	JSON_KEY_USERNAME   = "username"
	JSON_KEY_USER_TOKEN = "token"

	JSON_KEY_USER_ID   = "user_id"
	JSON_KEY_MEMBER_ID = "user_id"

	JSON_KEY_COMPANY_NAME    = "company_name"
	JSON_KEY_COMPANY_CONTACT = "company_contact"
	JSON_KEY_USER_PERMISSION = "user_permission"
)

func UserSignInHandler(c *gin.Context) {
	defer trace("UserSignInHandler")()

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	user_token := strings.TrimSpace(data.Get(JSON_KEY_USER_TOKEN).MustString())
	if len(user_token) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: "User token missing"})
		return
	}

	app_id := "313445519010095"
	app_secret := "c01e7696a4dc07ac4e2be87f867c9348"

	app := fb.New(app_id, app_secret)

	// exchange the short-term token to a long lived token(this synchronously calls facebook!!!)
	app_token, _, err := app.ExchangeToken(user_token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	res, err := fb.Get("me", fb.Params{
		"access_token": app_token,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	var username, fb_id string
	var v interface{}
	var ok bool

	if v, ok = res["name"]; ok {
		username, ok = v.(string)
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: "error facebook response: username field missing"})
		return
	}

	if v, ok = res["id"]; ok {
		fb_id, ok = v.(string)
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: "error facebook response: facebook_id field missing"})
		return
	}

	username = strings.TrimSpace(username)
	fb_id = strings.TrimSpace(fb_id)

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}
	defer func() {
		if err != nil && tnx != nil {
			tnx.Rollback()
		}
	}()

	var user *models.User
	if user, err = Store.FindUserWithProviderIdInTx(tnx,
		models.AUTH_PROVIDER_FACEBOOK, fb_id); err != nil {
		if err != models.ErrNoData {
			c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
			return
		} else {
			// the user doesn't exist, so try inserting the user
			new_user := &models.User{Username: username,
				ProviderID:     models.AUTH_PROVIDER_FACEBOOK,
				UserProviderID: fb_id}
			user, err = Store.CreateUserInTx(tnx, new_user)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
				return
			}
			tnx.Commit()
			tnx = nil
		}
	}
	tnx = nil

	// log-in the user for subsequent requests
	auth.LoginUser(user, c.Writer)

	c.JSON(http.StatusOK, map[string]interface{}{
		JSON_KEY_USERNAME: username,
		JSON_KEY_USER_ID:  user.UserId,
	})
}

// lists the companies this user belongs in
func UserCompanyListHandler(c *gin.Context) {
	defer trace("UserCompanyListHandler")()

	current_user, err := auth.GetCurrentUser(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{ERROR_MSG: err.Error()})
		return
	}

	company_permissions, err := Store.GetUserCompanyPermissions(current_user)
	if err != nil && err != models.ErrNoData {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG: err.Error()})
		return
	}

	/**
	 * See link why we don't do {@code var companies []interface{})
	 * https://danott.co/posts/json-marshalling-empty-slices-to-empty-arrays-in-go.html
	 */
	companies := make([]interface{}, 0)
	for i := 0; i < len(company_permissions); i++ {
		company := make(map[string]interface{})

		company[JSON_KEY_COMPANY_ID] = company_permissions[i].
			CompanyInfo.CompanyId
		company[JSON_KEY_COMPANY_NAME] = company_permissions[i].
			CompanyInfo.CompanyName
		company[JSON_KEY_USER_PERMISSION] = company_permissions[i].
			Permission.EncodedPermission

		companies = append(companies, company)
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"companies": companies,
	})
}
