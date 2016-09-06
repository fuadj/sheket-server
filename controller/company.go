package controller

import (
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"net/http"
	"sheket/server/controller/auth"
	sh "sheket/server/controller/sheket_handler"
	"sheket/server/models"
	"strings"
)

func AddCompanyMember(c *gin.Context) *sh.SheketError {
	defer trace("AddCompanyMember")()

	info, err := GetIdentityInfo(c.Request)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}
	company_id := info.CompanyId

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}

	invalid_id := int64(-1)
	member_id := data.Get(JSON_KEY_USER_ID).MustInt64(invalid_id)
	encoded_permission := strings.TrimSpace(data.Get(JSON_KEY_USER_PERMISSION).MustString())

	if member_id == invalid_id ||
		len(encoded_permission) == 0 {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: "error parsing member id"}
	}

	p := &models.UserPermission{}

	p.EncodedPermission = encoded_permission
	p.CompanyId = company_id
	p.UserId = member_id

	member, err := Store.FindUserById(member_id)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: "Couldn't find member: '" + err.Error() + "'"}
	}

	tnx, err := Store.Begin()
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	rev := &models.ShEntityRevision{
		CompanyId:        company_id,
		EntityType:       models.REV_ENTITY_MEMBERS,
		ActionType:       models.REV_ACTION_CREATE,
		EntityAffectedId: member_id,
		AdditionalInfo:   -1,
	}

	_, err = Store.AddEntityRevisionInTx(tnx, rev)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}
	tnx.Commit()

	c.JSON(http.StatusOK, map[string]interface{}{
		JSON_KEY_MEMBER_ID: member_id,
		JSON_KEY_USERNAME:  member.Username,
	})

	return nil
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
