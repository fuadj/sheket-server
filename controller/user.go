package controller

import (
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	fb "github.com/huandu/facebook"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"os"
	"sheket/server/controller/auth"
	sh "sheket/server/controller/sheket_handler"
	"sheket/server/models"
	sp "sheket/server/sheketproto"
	"strings"
	"fmt"
)

const (
	JSON_KEY_USERNAME   = "username"
	JSON_KEY_USER_TOKEN = "token"
	// this is an optional parameter, it should be included
	// and its value set to true if the app is a SheketPay client.
	// If this is set, only users who are AUTHORIZED will be
	// given login cookie.
	JSON_KEY_IS_SHEKET_PAY = "is_sheket_pay"

	JSON_KEY_USER_ID   = "user_id"
	JSON_KEY_MEMBER_ID = "user_id"

	JSON_KEY_COMPANY_NAME    = "company_name"
	JSON_KEY_COMPANY_CONTACT = "company_contact"
	JSON_KEY_USER_PERMISSION = "user_permission"

	JSON_KEY_NEW_USER_NAME = "new_user_name"
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

	var username, fb_id string
	var v interface{}
	var ok bool

	if v, ok = res["name"]; ok {
		username, ok = v.(string)
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

	return strings.TrimSpace(username), strings.TrimSpace(fb_id), nil
}

func (s *SheketController) UserSignup(c context.Context, request *sp.SingupRequest) (response *sp.SignupResponse, err error) {
	defer trace("UserSignup")()

	fb_id, username, err := getFacebookIdAndName(request)
	if err != nil {
		return nil, err
	}

	tnx, err := Store.GetDataStore().Begin()
	if err != nil {
		return nil, err
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
			return nil, err
		} else { // err == models.ErrNoData( which means user doesn't exist), create it
			new_user := &models.User{Username: username,
				ProviderID:     models.AUTH_PROVIDER_FACEBOOK,
				UserProviderID: fb_id}
			user, err = Store.CreateUserInTx(tnx, new_user)
			if err != nil {
				return nil, err
			}
			tnx.Commit()
			tnx = nil
		}
	}
	tnx = nil

	response = new(sp.SignupResponse)
	response.LoginCookie, err = auth.GenerateLoginCookie(user)
	if err != nil {
		return nil, err
	}

	response.Username = user.Username

	return response, nil
}

func (s *SheketController) SyncCompanies(c context.Context, request *sp.SyncCompanyRequest) (response *sp.CompanyList, err error) {
	defer trace("SyncCompanies")()

	user, err := auth.GetUser(request.Auth.LoginCookie)
	if err != nil {
		return nil, err
	}

	company_permissions, err := Store.GetUserCompanyPermissions(user)
	if err != nil && err != models.ErrNoData {
		return nil, err
	}

	user_companies := new(sp.CompanyList)

	for i := 0; i < len(company_permissions); i++ {
		company := new(sp.Company)

		company.CompanyId = company_permissions[i].CompanyInfo.CompanyId
		company.CompanyName = company_permissions[i].CompanyInfo.CompanyName
		company.Permission = company_permissions[i].Permission.EncodedPermission

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

func EditUserNameHandler(c *gin.Context) *sh.SheketError {
	defer trace("EditUserNameHandler")()

	current_user, err := auth.GetCurrentUser(c.Request)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}

	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	new_user_name := data.Get(JSON_KEY_NEW_USER_NAME).MustString("")
	if new_user_name == "" {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: "invalid username"}
	}

	tnx, err := Store.Begin()
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	current_user.Username = new_user_name
	_, err = Store.UpdateUserInTx(tnx, current_user)
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

