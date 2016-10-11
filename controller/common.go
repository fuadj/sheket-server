package controller

import (
	"sheket/server/controller/auth"
	"sheket/server/models"
	"sheket/server/sheketproto"
)

var Store models.ShStore

type UserCompanyPermission struct {
	CompanyId  int
	User       *models.User
	Permission *models.UserPermission
}

func GetUserWithCompanyPermission(companyAuth *sheketproto.CompanyAuth) (*UserCompanyPermission, error) {
	user, err := auth.GetUser(companyAuth.SheketAuth.LoginCookie)
	if err != nil {
		return nil, err
	}

	permission, err := Store.GetUserPermission(user, int(companyAuth.CompanyId.CompanyId))
	if err != nil {
		return nil, err
	}

	return &UserCompanyPermission{
		CompanyId:  int(companyAuth.CompanyId.CompanyId),
		User:       user,
		Permission: permission}, nil
}
