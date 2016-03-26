package controller

import (
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"net/http"
	"sheket/server/controller/auth"
	"sheket/server/models"
	"strings"
	"fmt"
)

const (
	key_username      = "username"
	key_password      = "password"
	key_new_user_id   = "new_user_id"
	key_login_status  = "login_status"
	key_login_message = "login_message"

	key_company_name    = "company_name"
	key_company_contact = "company_contact"
	key_new_company_id  = "new_company_id"
)

func UserSignupHandler(w http.ResponseWriter, r *http.Request) {
	defer trace("UserSignupHandler")()

	data, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	invalid_user_name := "111_invalid"

	username := data.Get(key_username).MustString(invalid_user_name)
	if strings.Compare(invalid_user_name, username) == 0 {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	password := data.Get(key_password).MustString()
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

	result := make(map[string]interface{}, 10)
	result[key_new_user_id] = created.UserId
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

func CompanyCreateHandler(w http.ResponseWriter, r *http.Request) {
	current_user, err := auth.GetCurrentUser(r)
	if err != nil {
		writeErrorResponse(w, http.StatusNonAuthoritativeInfo)
		return
	}

	data, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	company_name := data.Get(key_company_name).MustString()
	if len(company_name) == 0 {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}
	contact := data.Get(key_company_contact).MustString()

	company := &models.Company{CompanyName: company_name, Contact: contact}
	created, err := Store.CreateCompany(current_user, company)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make(map[string]interface{}, 10)

	result[key_new_company_id] = created.CompanyId
	result[key_company_name] = company_name
	result[key_company_contact] = contact

	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}

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

	username := data.Get(key_username).MustString()
	password := data.Get(key_password).MustString()
	if len(username) == 0 ||
		len(password) == 0 {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	user := &models.User{Username: username, HashedPassword: auth.HashPassword(password)}
	logged_user, err := auth.AuthenticateUser(user, password)
	result := make(map[string]interface{}, 10)
	if err == nil {
		auth.LoginUser(logged_user, w)
		result[key_login_status] = true
		result[key_login_message] = "login successful"
	} else {
		result[key_login_status] = false
		result[key_login_message] = "Incorrect username password combination!"
	}
	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
