package controller

import (
	fb "github.com/huandu/facebook"
	"golang.org/x/net/context"
	"log"
	"os"
	"sheket/server/controller/auth"
	"sheket/server/models"
	sp "sheket/server/sheketproto"
	"strings"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
)

var fb_app_secret string

func init() {
	if fb_app_secret = os.Getenv("FB_APP_SECRET"); fb_app_secret == "" {
		log.Fatal("$FB_APP_SECRET must be set")
	}
}

func getFacebookIdAndName(request *sp.SingupRequest) (fb_id, fb_name string, err error) {
	user_token := request.Token
	app_id := "313445519010095"

	app := fb.New(app_id, fb_app_secret)

	// exchange the short-term token to a long lived token(this synchronously calls facebook!!!)
	app_token, _, err := app.ExchangeToken(user_token)
	if err != nil {
		return fb_id, fb_name, err
	}

	res, err := fb.Get("me", fb.Params{
		"access_token": app_token,
	})
	if err != nil {
		return fb_id, fb_name, err
	}

	var v interface{}
	var ok bool

	if v, ok = res["name"]; ok {
		fb_name, ok = v.(string)
		if !ok {
			return fb_id, fb_name, fmt.Errorf("error facebook response: username field missing")
		}
	}

	if v, ok = res["id"]; ok {
		fb_id, ok = v.(string)
		if !ok {
			return fb_id, fb_name, fmt.Errorf("error facebook response: facebook_id field missing")
		}
	}

	return strings.TrimSpace(fb_name), strings.TrimSpace(fb_id), nil
}

func (s *SheketController) UserSignup(c context.Context, request *sp.SingupRequest) (response *sp.SignupResponse, err error) {
	defer trace("UserSignup")()

	//fb_id, username, err := getFacebookIdAndName(request)
	fb_id := "1417001148315681"
	username := "abcd"
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	defer func() {
		if err != nil && tnx != nil {
			tnx.Rollback()
		}
	}()

	var user *models.User
	if user, err = Store.FindUserWithProviderIdInTx(tnx,
		models.AUTH_PROVIDER_FACEBOOK, fb_id); err != nil {

		if err != models.ErrNoData {
			return nil, grpc.Errorf(codes.Internal, "%v", err)
		} else { // err == models.ErrNoData( which means user doesn't exist), create it
			new_user := &models.User{Username: username,
				ProviderID:     models.AUTH_PROVIDER_FACEBOOK,
				UserProviderID: fb_id}
			user, err = Store.CreateUserInTx(tnx, new_user)
			if err != nil {
				return nil, grpc.Errorf(codes.Internal, "%v", err)
			}
			tnx.Commit()
			tnx = nil
		}
	}
	tnx = nil

	response = new(sp.SignupResponse)
	response.UserId = int32(user.UserId)
	response.LoginCookie, err = auth.GenerateLoginCookie(user)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	response.Username = user.Username

	return response, nil
}

func (s *SheketController) SyncCompanies(c context.Context, request *sp.SyncCompanyRequest) (response *sp.CompanyList, err error) {
	defer trace("SyncCompanies")()

	user, err := auth.GetUser(request.Auth.LoginCookie)
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
	}

	company_permissions, err := Store.GetUserCompanyPermissions(user)
	if err != nil && err != models.ErrNoData {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	user_companies := new(sp.CompanyList)

	for i := 0; i < len(company_permissions); i++ {
		company := new(sp.Company)

		company.CompanyId = int32(company_permissions[i].CompanyInfo.CompanyId)
		company.CompanyName = company_permissions[i].CompanyInfo.CompanyName
		company.Permission = company_permissions[i].Permission.EncodedPermission
		company.PaymentId = generatePaymentId(&company_permissions[i].CompanyInfo)

		license, err := GenerateCompanyLicense(
			company_permissions[i].CompanyInfo.CompanyId,
			user.UserId,
			company_permissions[i].CompanyInfo.EncodedPayment,
			request.DeviceId, request.LocalUserTime)

		if err != nil {
			license = ""
		}

		company.SignedLicense = license

		user_companies.Companies = append(user_companies.Companies, company)
	}

	return user_companies, nil
}

func (s *SheketController) EditUserName(c context.Context, request *sp.EditUserNameRequest) (response *sp.EmptyResponse, err error) {
	defer trace("EditUserName")()

	user, err := auth.GetUser(request.Auth.LoginCookie)
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
	}

	tnx, err := Store.Begin()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	user.Username = request.NewName
	_, err = Store.UpdateUserInTx(tnx, user)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	err = tnx.Commit()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	response = new(sp.EmptyResponse)
	return response, nil
}
