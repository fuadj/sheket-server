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

func AddCompanyMember(w http.ResponseWriter, r *http.Request) {
	defer trace("AddCompanyMember")()

	company_id := GetCurrentCompanyId(r)
	if company_id == INVALID_COMPANY_ID {
		writeErrorResponse(w, http.StatusNonAuthoritativeInfo)
		return
	}

	if _, err := currentUserGetter(r); err != nil {
		writeErrorResponse(w, http.StatusNonAuthoritativeInfo, err.Error())
		return
	}

	data, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	invalid_id := int64(-1)
	member_id := data.Get(JSON_KEY_USER_ID).MustInt64(invalid_id)
	encoded_permission := strings.TrimSpace(data.Get(JSON_KEY_USER_PERMISSION).MustString())

	if member_id == invalid_id ||
		len(encoded_permission) == 0 {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	p := &models.UserPermission{}

	p.EncodedPermission = encoded_permission
	p.CompanyId = company_id
	p.UserId = member_id

	member, err := Store.FindUserById(member_id)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error() + "e0")
		return
	}

	tnx, err := Store.Begin()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error() + "e1")
		return
	}

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error() + "e2")
		return
	}

	rev := &models.ShEntityRevision{
		CompanyId:company_id,
		EntityType:models.REV_ENTITY_MEMBERS,
		ActionType:models.REV_ACTION_CREATE,
		EntityAffectedId:member_id,
		AdditionalInfo:-1,
	}

	_, err = Store.AddEntityRevisionInTx(tnx, rev)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error() + "e3")
		return
	}
	tnx.Commit()

	result := map[string]interface{}{
		JSON_KEY_MEMBER_ID: member_id,
		JSON_KEY_USERNAME: member.Username,
	}
	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func CompanyCreateHandler(w http.ResponseWriter, r *http.Request) {
	defer trace("CompanyCreateHandler")()

	current_user, err := auth.GetCurrentUser(r)
	if err != nil {
		writeErrorResponse(w, http.StatusNonAuthoritativeInfo, "e1")
		return
	}

	data, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "e2")
		return
	}

	company_name := data.Get(JSON_KEY_COMPANY_NAME).MustString()
	if len(company_name) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "e3")
		return
	}
	contact := data.Get(JSON_KEY_COMPANY_CONTACT).MustString()

	company := &models.Company{CompanyName: company_name, Contact: contact}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error() + "e4")
		return
	}
	created, err := Store.CreateCompanyInTx(tnx, current_user, company)
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, err.Error() + "e5")
		return
	}

	result := make(map[string]interface{}, 10)

	result[JSON_KEY_COMPANY_ID] = created.CompanyId
	result[JSON_KEY_COMPANY_NAME] = company_name
	result[JSON_KEY_COMPANY_CONTACT] = contact

	p := &models.UserPermission{CompanyId:created.CompanyId,
		UserId:current_user.UserId,
		PermissionType:models.PERMISSION_TYPE_CREATOR}
	encoded := p.Encode()
	result[JSON_KEY_USER_PERMISSION] = encoded

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, err.Error() + "e6")
		return
	}

	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, "e7")
		return
	}
	tnx.Commit()

	w.WriteHeader(http.StatusOK)
	w.Write(b)
	fmt.Printf("%s", string(b))
}

