package controller

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"net/http"
	"sheket/server/controller/auth"
	"sheket/server/models"
	"strings"
)

const (
	JSON_KEY_USERNAME = "username"
	JSON_KEY_PASSWORD = "password"

	JSON_KEY_USER_ID       = "user_id"
	JSON_KEY_MEMBER_ID     = "user_id"

	JSON_KEY_COMPANY_NAME    = "company_name"
	JSON_KEY_COMPANY_CONTACT = "company_contact"
	JSON_KEY_USER_PERMISSION = "user_permission"
)

func UserSignupHandler(c *gin.Context) {
	defer trace("UserSignupHandler")()

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	invalid_user_name := "111_invalid"

	username := data.Get(JSON_KEY_USERNAME).MustString(invalid_user_name)
	if strings.Compare(invalid_user_name, username) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	password := data.Get(JSON_KEY_PASSWORD).MustString()
	if len(password) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	prev_user, err := Store.FindUserByNameInTx(tnx, username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}
	if prev_user != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: fmt.Sprintf("%s already exists", username)})
		return
	}

	user := &models.User{Username: username,
		HashedPassword: auth.HashPassword(password)}
	created, err := Store.CreateUserInTx(tnx, user)
	if err != nil {
		tnx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}
	tnx.Commit()

	// log-in the user for subsequent requests
	auth.LoginUser(created, c.Writer)

	c.JSON(http.StatusOK, map[string]interface{}{
		JSON_KEY_USERNAME: username,
		JSON_KEY_USER_ID:  created.UserId,
	})
}

func UserLoginHandler(c *gin.Context) {
	defer trace("UserLoginHandler")()

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	username := data.Get(JSON_KEY_USERNAME).MustString()
	password := data.Get(JSON_KEY_PASSWORD).MustString()
	if len(username) == 0 ||
		len(password) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	user := &models.User{Username: username, HashedPassword: auth.HashPassword(password)}
	auth_user, err := auth.AuthenticateUser(user, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: "Incorrect username password combination!"})
		return
	}
	auth.LoginUser(auth_user, c.Writer)
	c.JSON(http.StatusOK, map[string]interface{}{
		JSON_KEY_USER_ID: auth_user.UserId,
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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG: err.Error()})
		return
	}

	var companies []interface{}

	for i := 0; i < len(company_permissions); i++ {
		company := make(map[string]interface{}, 10)

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
