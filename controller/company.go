package controller

import (
	"github.com/gin-gonic/gin"
	"sheket/server/controller/auth"
	sh "sheket/server/controller/sheket_handler"
	"sheket/server/models"
	"strings"
	"golang.org/x/net/context"
	sp "sheket/server/sheketproto"
	"fmt"
	"time"
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
		UserId:int(request.EmployeeId),
	}

	member, err := Store.FindUserById(p.UserId)
	if err != nil {
		return nil, fmt.Errorf("Couldn't find employee:'%v'", err.Error())
	}

	tnx, err := Store.Begin()
	if err != nil {
		return nil, err
	}

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	tnx.Commit()

	response = new(sp.AddEmployeeResponse)
	response.EmployeeId = int32(p.UserId)
	response.EmployeeName = member.Username

	return response, nil
}

func getSingleUserContract() string {
	payment_info := &models.PaymentInfo{}

	payment_info.ContractType = models.PAYMENT_CONTRACT_TYPE_SINGLE_USE
	payment_info.EmployeeLimit = _to_server_limit(CLIENT_NO_LIMIT)
	payment_info.BranchLimit = _to_server_limit(CLIENT_NO_LIMIT)
	payment_info.ItemLimit = _to_server_limit(CLIENT_NO_LIMIT)
	payment_info.DurationInDays = 60 		// these is in days(2 months)

	payment_info.IssuedDate = time.Now().Unix()

	return payment_info.Encode()
}

func generatePaymentId(company *models.Company) string {
	// TODO: make it robust, add error detection
	return fmt.Sprintf("%d", company.CompanyId)
}

func (s *SheketController) CreateCompany(c context.Context, request *sp.NewCompanyRequest) (response *sp.Company, err error) {
	defer trace("CreateCompany")()

	user, err := auth.GetUser(request.Auth.LoginCookie)
	if err != nil {
		return nil, err
	}

	// TODO: update the initial contract type
	payment := getSingleUserContract()

	company := &models.Company{
		CompanyName:request.CompanyName,
		EncodedPayment:payment,
	}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		return nil, err
	}

	created_company, err := Store.CreateCompanyInTx(tnx, user, company)
	if err != nil {
		tnx.Rollback()
		return nil, err
	}

	permission := &models.UserPermission{CompanyId: created_company.CompanyId,
		UserId:         user.UserId,
		PermissionType: models.PERMISSION_TYPE_OWNER}

	_, err = Store.SetUserPermissionInTx(tnx, permission)
	if err != nil {
		tnx.Rollback()
		return nil, err
	}
	tnx.Commit()


	license, err := GenerateCompanyLicense(
		created_company.CompanyId,
		user.UserId,
		payment,
		request.DeviceId, request.LocalUserTime)
	if err != nil {
		return nil, err
	}

	response = new(sp.Company)
	response.CompanyId = int32(created_company.CompanyId)
	response.CompanyName = request.CompanyName
	response.Permission = permission.Encode()
	response.SignedLicense = license
	response.PaymentId = generatePaymentId(created_company)

	return response, nil
}

func EditCompanyNameHandler(c *gin.Context) *sh.SheketError {
	/*
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

	*/
	return nil
}
