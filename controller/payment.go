package controller

import (
	"github.com/gin-gonic/gin"
	"sheket/server/models"
	"github.com/bitly/go-simplejson"
	"time"
	"fmt"
	"net/http"
)

const (
	CLIENT_NO_LIMIT int64 = -1
)

/**
 * Payment for a license to use Sheket is issued here. This ROUTE needs
 * to be made more secure as it is the only place users will pay for the service.
 * If payment succeeds, the company will have a receipt that will be valid for
 * contract duration.
 *
 * TODO: currently it overwrites the payment, in the future "see" what has already
 * been paid and do {extend | upgrade}
 */
func IssuePaymentHandler(c *gin.Context) {
	data, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	values, err := data.Map()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: err.Error()})
		return
	}

	payment_info := models.PaymentInfo{}

	var company_id int64
	var not_ok bool
	var missing_field string

	// These cascade of if statements will only be executed of there was no problem
	// If a problem is encountered, that if condition will be fulfilled, so the rest will basically stop
	if company_id, not_ok = get_int64(models.PAYMENT_JSON_CONTRACT_TYPE, values, nil); not_ok {
		missing_field = models.PAYMENT_JSON_COMPANY_ID
	} else if payment_info.ContractType, not_ok = get_int64(models.PAYMENT_JSON_CONTRACT_TYPE, values, nil); not_ok {
		missing_field = models.PAYMENT_JSON_CONTRACT_TYPE
	} else if payment_info.DurationInDays, not_ok = get_int64(models.PAYMENT_JSON_DURATION, values, nil); not_ok {
		missing_field = models.PAYMENT_JSON_DURATION
	} else if payment_info.EmployeeLimit, not_ok = get_int64(models.PAYMENT_JSON_LIMIT_EMPLOYEE, values, nil); not_ok {
		missing_field = models.PAYMENT_JSON_LIMIT_EMPLOYEE
	} else if payment_info.BranchLimit, not_ok = get_int64(models.PAYMENT_JSON_LIMIT_BRANCH, values, nil); not_ok {
		missing_field = models.PAYMENT_JSON_LIMIT_BRANCH
	} else if payment_info.ItemLimit, not_ok = get_int64(models.PAYMENT_JSON_LIMIT_ITEM, values, nil); not_ok {
		missing_field = models.PAYMENT_JSON_LIMIT_ITEM
	}

	// we can check here if the above was successful or not
	if not_ok {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG: fmt.Sprintf("missing: %s", missing_field)})
		return
	}

	if payment_info.EmployeeLimit == CLIENT_NO_LIMIT {
		payment_info.EmployeeLimit = models.PAYMENT_LIMIT_NONE
	}
	if payment_info.BranchLimit == CLIENT_NO_LIMIT {
		payment_info.BranchLimit = models.PAYMENT_LIMIT_NONE
	}
	if payment_info.ItemLimit == CLIENT_NO_LIMIT {
		payment_info.ItemLimit = models.PAYMENT_LIMIT_NONE
	}

	payment_info.IssuedDate = time.Now().Unix()

	company, err := Store.GetCompanyById(company_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG: err.Error()})
		return
	}
	company.EncodedPayment = payment_info.Encode()
	tnx, err := Store.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG: err.Error()})
		return
	}

	company, err = Store.UpdateCompanyInTx(tnx, company)
	if err != nil {
		tnx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG: err.Error()})
		return
	}
	tnx.Commit()

	c.JSON(http.StatusOK, map[string]interface{}{
		JSON_KEY_COMPANY_ID: company_id,
		"payment": fmt.Sprintf("Successful payment for %d days", payment_info.DurationInDays),
	})
}

/**
 * In the current implementation, users aren't allowed to make payment directly due to the
 * non-integration with payment services like HelloCash and M-Birr.
 * Payment happens through "agents". After an agent has issued a payment request for a user's
 * company, then the user needs to verify the payment has been made to continue using the app.
 * This is particularly necessary for uses that don't use the sync feature as the payment verification
 * should always happen on every sync.
 */
func VerifyPaymentHandler(c *gin.Context) {

}