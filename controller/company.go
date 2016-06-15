package controller

import (
	"github.com/bitly/go-simplejson"
	"net/http"
	"sheket/server/controller/auth"
	"sheket/server/models"
	"strings"
	"github.com/gin-gonic/gin"
)

func AddCompanyMember(c *gin.Context) {
	defer trace("AddCompanyMember")()

	company_id := GetCurrentCompanyId(c.Request)
	if company_id == INVALID_COMPANY_ID {
		c.String(http.StatusUnauthorized, "")
		return
	}

	if _, err := currentUserGetter(c.Request); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{ERROR_MSG:err.Error()})
		return
	}

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG:err.Error()})
		return
	}

	invalid_id := int64(-1)
	member_id := data.Get(JSON_KEY_USER_ID).MustInt64(invalid_id)
	encoded_permission := strings.TrimSpace(data.Get(JSON_KEY_USER_PERMISSION).MustString())

	if member_id == invalid_id ||
		len(encoded_permission) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG:""})
		return
	}

	p := &models.UserPermission{}

	p.EncodedPermission = encoded_permission
	p.CompanyId = company_id
	p.UserId = member_id

	member, err := Store.FindUserById(member_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:""})
		return
	}

	tnx, err := Store.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:""})
		return
	}

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:""})
		return
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
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:""})
		return
	}
	tnx.Commit()

	c.JSON(http.StatusOK, map[string]interface{}{
		JSON_KEY_MEMBER_ID: member_id,
		JSON_KEY_USERNAME:  member.Username,
	})
}

func CompanyCreateHandler(c *gin.Context) {
	defer trace("CompanyCreateHandler")()

	current_user, err := auth.GetCurrentUser(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{ERROR_MSG:""})
		return
	}

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG:err.Error()})
		return
	}

	company_name := data.Get(JSON_KEY_COMPANY_NAME).MustString()
	if len(company_name) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG:"Empty company name"})
		return
	}
	contact := data.Get(JSON_KEY_COMPANY_CONTACT).MustString()

	company := &models.Company{CompanyName: company_name, Contact: contact}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:err.Error()})
		return
	}
	created, err := Store.CreateCompanyInTx(tnx, current_user, company)
	if err != nil {
		tnx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:err.Error()})
		return
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
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:err.Error()})
		return
	}
	tnx.Commit()

	c.JSON(http.StatusOK, result)
}
