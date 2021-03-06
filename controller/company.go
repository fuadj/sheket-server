package controller

import (
	"fmt"
	"golang.org/x/net/context"
	"sheket/server/controller/auth"
	"sheket/server/models"
	sp "sheket/server/sheketproto"
	"strings"
	"time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *SheketController) AddEmployee(c context.Context, request *sp.AddEmployeeRequest) (response *sp.AddEmployeeResponse, err error) {
	defer trace("AddEmployee")()

	user_info, err := GetUserWithCompanyPermission(request.CompanyAuth)
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
	}

	request.Permission = strings.TrimSpace(request.Permission)
	if len(request.Permission) == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "%v", err)
	}

	p := &models.UserPermission{
		CompanyId:         user_info.CompanyId,
		EncodedPermission: request.Permission,
		UserId:            int(request.EmployeeId),
	}

	member, err := Store.FindUserById(p.UserId)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	tnx, err := Store.Begin()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	_, err = Store.SetUserPermissionInTx(tnx, p)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
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
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	tnx.Commit()

	response = new(sp.AddEmployeeResponse)
	response.EmployeeId = int32(p.UserId)
	response.EmployeeName = member.Username

	return response, nil
}

func getSingleUserContract() string {
	payment_info := &models.PaymentInfo{}

	payment_info.ContractType = models.PAYMENT_CONTRACT_LIMITED_FREE
	payment_info.EmployeeLimit = _to_server_limit(CLIENT_NO_LIMIT)
	payment_info.BranchLimit = _to_server_limit(CLIENT_NO_LIMIT)
	payment_info.ItemLimit = _to_server_limit(CLIENT_NO_LIMIT)
	payment_info.DurationInDays = 30 // these is in days(1 month)

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
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
	}

	// TODO: update the initial contract type
	payment := getSingleUserContract()

	company := &models.Company{
		CompanyName:    request.CompanyName,
		EncodedPayment: payment,
	}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	created_company, err := Store.CreateCompanyInTx(tnx, user, company)
	if err != nil {
		tnx.Rollback()
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	permission := &models.UserPermission{CompanyId: created_company.CompanyId,
		UserId:         user.UserId,
		PermissionType: models.PERMISSION_TYPE_OWNER}
	permission.Encode()

	_, err = Store.SetUserPermissionInTx(tnx, permission)
	if err != nil {
		tnx.Rollback()
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	tnx.Commit()

	license, err := GenerateCompanyLicense(
		created_company.CompanyId,
		user.UserId,
		payment,
		request.DeviceId, request.LocalUserTime)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	response = new(sp.Company)
	response.CompanyId = int32(created_company.CompanyId)
	response.CompanyName = request.CompanyName
	response.Permission = permission.Encode()
	response.SignedLicense = license
	response.PaymentId = generatePaymentId(created_company)

	return response, nil
}

func (s *SheketController) EditCompany(c context.Context, request *sp.EditCompanyRequest) (response *sp.EmptyResponse, err error) {
	defer trace("EditCompany")()

	user_info, err := GetUserWithCompanyPermission(request.CompanyAuth)
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
	}
	company, err := Store.GetCompanyById(user_info.CompanyId)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	tnx, err := Store.Begin()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	company.CompanyName = request.NewName
	if _, err = Store.UpdateCompanyInTx(tnx, company); err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	if err = tnx.Commit(); err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	return &sp.EmptyResponse{}, nil
}
