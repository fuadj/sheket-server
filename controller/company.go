package controller

import (
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"net/http"
	"sheket/server/controller/auth"
	sh "sheket/server/controller/sheket_handler"
	"sheket/server/models"
	"strings"
	"golang.org/x/net/context"
	sp "sheket/server/sheketproto"
	"fmt"
)

const (
	JSON_KEY_NEW_COMPANY_NAME = "new_company_name"
)

func (s *SheketController) AddEmployee(c context.Context, request *sp.AddEmployeeRequest) (response *sp.AddEmployeeResponse, err error) {
	defer trace("AddEmployee")()

	user_info, err := GetUserWithCompanyPermission(request.CompanyAuth)
	if err != nil {
		return nil, err
	}

	request.Permission = strings.TrimSpace(request.Permission)
	if len(request.Permission) == 0 {
		return nil, fmt.Errorf("Invalid company permission")
	}

	p := &models.UserPermission{
		CompanyId:user_info.CompanyId,
		EncodedPermission:request.Permission,
		UserId:request.EmployeeId,
	}

	member, err := Store.FindUserById(p.UserId)
	if err != nil {
		return nil, fmt.Errorf("Couldn't find employee:'%v'", err.Error())
	}

	tnx, err := Store.Begin()
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	rev := &models.ShEntityRevision{
		CompanyId:        user_info.CompanyId,
		EntityType:       models.REV_ENTITY_MEMBERS,
		ActionType:       models.REV_ACTION_CREATE,
		EntityAffectedId: p.UserId,
		AdditionalInfo:   -1,
	}

	_, err = Store.AddEntityRevisionInTx(tnx, rev)
	if err != nil {
		tnx.Rollback()
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}
	tnx.Commit()

	response = new(sp.AddEmployeeResponse)
	response.EmployeeId = p.UserId
	response.EmployeeName = member.Username

	return response, nil
}

func CompanyCreateHandler(c *gin.Context) *sh.SheketError {
	defer trace("CompanyCreateHandler")()

	current_user, err := auth.GetCurrentUser(c.Request)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}

	company_name := data.Get(JSON_KEY_COMPANY_NAME).MustString()
	if len(company_name) == 0 {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: "Empty company name"}
	}
	contact := data.Get(JSON_KEY_COMPANY_CONTACT).MustString()

	company := &models.Company{CompanyName: company_name, Contact: contact}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}
	created, err := Store.CreateCompanyInTx(tnx, current_user, company)
	if err != nil {
		tnx.Rollback()
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	result := make(map[string]interface{}, 10)

	result[JSON_KEY_COMPANY_ID] = created.CompanyId
	result[JSON_KEY_COMPANY_NAME] = company_name
	result[JSON_KEY_COMPANY_CONTACT] = contact

	p := &models.UserPermission{CompanyId: created.CompanyId,
		UserId:         current_user.UserId,
		PermissionType: models.PERMISSION_TYPE_CREATOR}
	encoded := p.Encode()
	result[JSON_KEY_USER_PERMISSION] = encoded

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		tnx.Rollback()
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}
	tnx.Commit()

	c.JSON(http.StatusOK, result)
	return nil
}

func EditCompanyNameHandler(c *gin.Context) *sh.SheketError {
	defer trace("EditCompanyNameHandler")()

	info, err := GetIdentityInfo(c.Request)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	new_company_name := data.Get(JSON_KEY_NEW_COMPANY_NAME).MustString("")
	if new_company_name == "" {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: "invalid company name"}
	}

	company, err := Store.GetCompanyById(info.CompanyId)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	tnx, err := Store.Begin()
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	company.CompanyName = new_company_name
	_, err = Store.UpdateCompanyInTx(tnx, company)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}
	err = tnx.Commit()
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	// we don't actually send any "useful" data, it is just to inform that it was successful
	c.String(http.StatusOK, "")

	return nil
}
