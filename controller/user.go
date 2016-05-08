package controller

import (
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
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
	JSON_KEY_LOGIN_STATUS  = "login_status"
	JSON_KEY_LOGIN_MESSAGE = "login_message"

	JSON_KEY_COMPANY_NAME    = "company_name"
	JSON_KEY_COMPANY_CONTACT = "company_contact"
	JSON_KEY_USER_PERMISSION = "user_permission"
)

func UserSignupHandler(w http.ResponseWriter, r *http.Request) {
	defer trace("UserSignupHandler")()

	data, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	invalid_user_name := "111_invalid"

	username := data.Get(JSON_KEY_USERNAME).MustString(invalid_user_name)
	if strings.Compare(invalid_user_name, username) == 0 {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	password := data.Get(JSON_KEY_PASSWORD).MustString()
	if len(password) == 0 {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	prev_user, err := Store.FindUserByNameInTx(tnx, username)
	if prev_user != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("%s already exists", username))
		return
	}
	user := &models.User{Username: username,
		HashedPassword: auth.HashPassword(password)}
	created, err := Store.CreateUserInTx(tnx, user)
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := map[string]interface{}{
		JSON_KEY_USERNAME: username,
		JSON_KEY_USER_ID:  created.UserId,
	}
	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}
	tnx.Commit()

	// log-in the user for subsequent requests
	auth.LoginUser(created, w)

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func UserLoginHandler(w http.ResponseWriter, r *http.Request) {
	defer trace("UserLoginHandler")()

	data, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	username := data.Get(JSON_KEY_USERNAME).MustString()
	password := data.Get(JSON_KEY_PASSWORD).MustString()
	if len(username) == 0 ||
		len(password) == 0 {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	user := &models.User{Username: username, HashedPassword: auth.HashPassword(password)}
	auth_user, err := auth.AuthenticateUser(user, password)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "Incorrect username password combination!")
		return
	}
	result := map[string]interface{}{
		JSON_KEY_USER_ID: auth_user.UserId,
	}
	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}

	auth.LoginUser(auth_user, w)

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// lists the companies this user belongs in
func UserCompanyListHandler(w http.ResponseWriter, r *http.Request) {
	defer trace("UserCompanyListHandler")()

	current_user, err := auth.GetCurrentUser(r)
	if err != nil {
		writeErrorResponse(w, http.StatusNonAuthoritativeInfo, "euser")
		return
	}

	company_permissions, err := Store.GetUserCompanyPermissions(current_user)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error()+"permission")
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
	result := map[string]interface{}{
		"companies": companies,
	}

	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
