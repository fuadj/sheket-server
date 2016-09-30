package controller

import (
	"fmt"
	"net/http"
	"sheket/server/controller/auth"
	"sheket/server/models"
	"sheket/server/sheketproto"
	"strconv"
)

const (
	INVALID_COMPANY_ID int64 = -1

	/**
	 * IMPORTANT: since this is used in a header it must comply with
	 * the Canonical form of a header. That is [a-zA-Z]+(?:-?[a-zA-Z]+)*
	 * You can't make the hyphen(-) an underscore. That is a BUG and
	 * causes the whole app to not work.
	 */
	JSON_KEY_COMPANY_ID = "company-id"

	KEY_JSON_ID_OLD = "o"
	KEY_JSON_ID_NEW = "n"
)

var Store models.ShStore

type UserCompanyPermission struct {
	CompanyId  int64
	User       *models.User
	Permission *models.UserPermission
}

func GetCurrentCompanyId(r *http.Request) int64 {
	id, err := strconv.ParseInt(r.Header.Get(JSON_KEY_COMPANY_ID), 10, 64)
	if err != nil {
		return INVALID_COMPANY_ID
	}
	return id
}

// used in testing
var currentUserGetter = auth.GetCurrentUser

func GetIdentityInfo(r *http.Request) (*UserCompanyPermission, error) {
	company_id := GetCurrentCompanyId(r)
	if company_id == INVALID_COMPANY_ID {
		return nil, fmt.Errorf("Invalid company id")
	}

	user, err := currentUserGetter(r)
	if err != nil {
		return nil, fmt.Errorf("Invalid user cookie '%s'", err.Error())
	}

	permission, err := Store.GetUserPermission(user, company_id)
	if err != nil { // the user doesn't have permission to post
		return nil, fmt.Errorf("User doesn't have permission, %s", err.Error())
	}

	info := &UserCompanyPermission{CompanyId: company_id, User: user, Permission: permission}
	return info, nil
}

func GetUserWithCompanyPermission(companyAuth *sheketproto.CompanyAuth) (*UserCompanyPermission, error) {
	user, err := auth.GetUser(companyAuth.SheketAuth.LoginCookie)
	if err != nil {
		return nil, err
	}

	permission, err := Store.GetUserPermission(user, companyAuth.CompanyId.CompanyId)
	if err != nil {
		return nil, err
	}

	return &UserCompanyPermission{
		CompanyId:  companyAuth.CompanyId.CompanyId,
		User:       user,
		Permission: permission}, nil
}
