package controller

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"net/http"
	sh "sheket/server/controller/sheket_handler"
	"sheket/server/controller/signature"
	"sheket/server/models"
	"time"
)

const (
	JSON_PAYMENT_DESCRIPTION = "payment_desc"

	JSON_PAYMENT_SIGNED_LICENSE = "signed_license"

	JSON_PAYMENT_DEVICE_ID       = "device_id"
	JSON_PAYMENT_LOCAL_USER_TIME = "local_user_time"
)

const (
	CLIENT_NO_LIMIT int64 = -1
)

func _to_server_limit(val int64) int64 {
	if val == CLIENT_NO_LIMIT {
		return models.PAYMENT_LIMIT_NONE
	}
	return val
}

func _to_client_limit(val int64) int64 {
	if val == models.PAYMENT_LIMIT_NONE {
		return CLIENT_NO_LIMIT
	}
	return val
}

/**
 * Payment for a license to use Sheket is issued here. This ROUTE needs
 * to be made more secure as it is the only place users will pay for the service.
 * If payment succeeds, the company will have a receipt that will be valid for
 * contract duration.
 *
 * TODO: currently it overwrites the payment, in the future "see" what has already
 * been paid and do {extend | upgrade}
 */
func IssuePaymentHandler(c *gin.Context) *sh.SheketError {
	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}

	values, err := data.Map()
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}

	payment_info := &models.PaymentInfo{}

	var company_id int64
	var ok bool
	var missing_field string

	// These cascade of if statements will only be executed of there was no problem
	// If a problem is encountered, that if condition will be fulfilled, so the rest will basically stop
	if company_id, ok = get_int64(models.PAYMENT_JSON_COMPANY_ID, values, nil); !ok {
		missing_field = models.PAYMENT_JSON_COMPANY_ID
	} else if payment_info.ContractType, ok = get_int64(models.PAYMENT_JSON_CONTRACT_TYPE, values, nil); !ok {
		missing_field = models.PAYMENT_JSON_CONTRACT_TYPE
	} else if payment_info.DurationInDays, ok = get_int64(models.PAYMENT_JSON_DURATION, values, nil); !ok {
		missing_field = models.PAYMENT_JSON_DURATION
	} else if payment_info.EmployeeLimit, ok = get_int64(models.PAYMENT_JSON_LIMIT_EMPLOYEE, values, nil); !ok {
		missing_field = models.PAYMENT_JSON_LIMIT_EMPLOYEE
	} else if payment_info.BranchLimit, ok = get_int64(models.PAYMENT_JSON_LIMIT_BRANCH, values, nil); !ok {
		missing_field = models.PAYMENT_JSON_LIMIT_BRANCH
	} else if payment_info.ItemLimit, ok = get_int64(models.PAYMENT_JSON_LIMIT_ITEM, values, nil); !ok {
		missing_field = models.PAYMENT_JSON_LIMIT_ITEM
	}

	// we can check here if the above was successful or not
	if !ok {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: fmt.Sprintf("missing: %s", missing_field)}
	}

	payment_info.EmployeeLimit = _to_server_limit(payment_info.EmployeeLimit)
	payment_info.BranchLimit = _to_server_limit(payment_info.BranchLimit)
	payment_info.ItemLimit = _to_server_limit(payment_info.ItemLimit)

	payment_info.IssuedDate = time.Now().Unix()

	company, err := Store.GetCompanyById(company_id)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}
	company.EncodedPayment = payment_info.Encode()
	tnx, err := Store.Begin()
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	company, err = Store.UpdateCompanyInTx(tnx, company)
	if err != nil {
		tnx.Rollback()
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}
	tnx.Commit()

	c.JSON(http.StatusOK, map[string]interface{}{
		JSON_KEY_COMPANY_ID:      company_id,
		JSON_PAYMENT_DESCRIPTION: fmt.Sprintf("Successful payment for %d days", payment_info.DurationInDays),
	})

	return nil
}

/** * In the current implementation, users aren't allowed to make payment directly due to the
 * non-integration with payment services like HelloCash and M-Birr.
 * Payment happens through "agents". After an agent has issued a payment request for a user's
 * company, then the user needs to verify the payment has been made to continue using the app.
 * This is particularly necessary for uses that don't use the sync feature as the payment verification
 * should always happen on every sync.
 */
func VerifyPaymentHandler(c *gin.Context) *sh.SheketError {
	info, err := GetIdentityInfo(c.Request)
	if err != nil {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: err.Error()}
	}
	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	device_id := data.Get(JSON_PAYMENT_DEVICE_ID).MustString("")
	user_local_time := data.Get(JSON_PAYMENT_LOCAL_USER_TIME).MustString("")

	if device_id == "" || user_local_time == "" {
		return &sh.SheketError{Code: http.StatusBadRequest, Error: "missing payment fields"}
	}

	company, err := Store.GetCompanyById(info.CompanyId)
	if err != nil {
		return &sh.SheketError{Code: http.StatusInternalServerError, Error: err.Error()}
	}

	license, err := GenerateCompanyLicense(info.CompanyId, info.User.UserId,
		company.EncodedPayment, device_id, user_local_time)

	if err != nil {
		err_msg := err.Error()
		if err = revokeCompanyLicense(info.CompanyId); err != nil {
			// append the revoke error
			err_msg = err_msg + ":" + err.Error()
		}
		return &sh.SheketError{Code: http.StatusPaymentRequired, Error: err_msg}
	}

	c.JSON(http.StatusOK,
		gin.H{JSON_PAYMENT_SIGNED_LICENSE: license})
	return nil
}

/**
 * Generates a signed license if there is still paid period left. This doesn't query the db, only uses
 * the info provided.
 */
func GenerateCompanyLicense(company_id, user_id int64, encoded_payment, device_id, user_local_time string) (string, error) {
	payment_info, err := models.DecodePayment(encoded_payment)
	if err != nil {
		return "", err
	}

	current_date := time.Now().Unix()
	end_date := time.Unix(payment_info.IssuedDate, 0).
		AddDate(0, 0, int(payment_info.DurationInDays)).Unix()

	var remaining_days int64

	payment_expired := false

	if current_date > end_date {
		payment_expired = true
	} else {
		remaining_days = int64(
			time.Unix(end_date, 0).Sub(time.Unix(current_date, 0)).
				Hours() / 24.0)
		// if there is < 1 day of payment remaining, revoke it b/c that will be encoded as 0 when converted to int
		if remaining_days < 1 {
			payment_expired = true
		}
	}

	if payment_expired {
		return "", fmt.Errorf("license expired")
	}

	// if we've reached here, it means the user has valid remaining payment
	contract := fmt.Sprintf(""+
		"device_id:%s;"+
		"user_id:%d;"+
		"company_id:%d;"+
		"server_date_issued:%d;"+
		"local_date_issued:%s;"+
		"duration:%d;"+
		"contract_type:%d;"+
		"employees:%d;"+
		"branches:%d;"+
		"items:%d",
		device_id, user_id, company_id,
		current_date, user_local_time,
		payment_info.DurationInDays, payment_info.ContractType,
		_to_client_limit(payment_info.EmployeeLimit),
		_to_client_limit(payment_info.BranchLimit),
		_to_client_limit(payment_info.ItemLimit),
	)

	signature, err := signature.SignBase64EncodeMessage(contract)
	if err != nil {
		return "", err
	}

	license := fmt.Sprintf("%s_||_%s", contract, signature)
	return license, nil
}

func revokeCompanyLicense(company_id int64) error {
	tnx, err := Store.Begin()
	if err != nil {
		return err
	}
	company, err := Store.GetCompanyById(company_id)
	if err != nil {
		return err
	}
	company.EncodedPayment = ""

	company, err = Store.UpdateCompanyInTx(tnx, company)
	if err != nil {
		tnx.Rollback()
		return err
	}
	tnx.Commit()
	return nil
}
