package controller

import (
	"fmt"
	"golang.org/x/net/context"
	"sheket/server/controller/auth"
	"sheket/server/controller/signature"
	"sheket/server/models"
	sp "sheket/server/sheketproto"
	"time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	CLIENT_NO_LIMIT int = -1
)

func _to_server_limit(val int) int {
	if val == CLIENT_NO_LIMIT {
		return models.PAYMENT_LIMIT_NONE
	}
	return val
}

func _to_client_limit(val int) int {
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
func (s *SheketController) IssuePayment(c context.Context, request *sp.IssuePaymentRequest) (response *sp.IssuePaymentResponse, err error) {
	user, err := auth.GetUser(request.SheketAuth.LoginCookie)
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err.Error())
	}

	if err := is_user_allowed_to_issue_payment(user); err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "You don't have authority to issue payment")
	}

	payment := &models.PaymentInfo{}
	payment.ContractType = int(request.ContractType)
	payment.DurationInDays = int(request.DurationDays)

	payment.EmployeeLimit = _to_server_limit(int(request.EmployeeLimit))
	payment.BranchLimit = _to_server_limit(int(request.BranchLimit))
	payment.ItemLimit = _to_server_limit(int(request.ItemLimit))

	payment.IssuedDate = time.Now().Unix()

	company_id := int(request.CompanyId)
	company, err := Store.GetCompanyById(company_id)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	company.EncodedPayment = payment.Encode()
	tnx, err := Store.Begin()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	company, err = Store.UpdateCompanyInTx(tnx, company)
	if err != nil {
		tnx.Rollback()
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	tnx.Commit()

	return &sp.IssuePaymentResponse{
		IssuedCompanyId:    request.CompanyId,
		PaymentDescription: fmt.Sprintf("Successful payment for %d days", request.DurationDays),
	}, nil
}

func is_user_allowed_to_issue_payment(user *models.User) error {
	return nil
}

/** * In the current implementation, users aren't allowed to make payment directly due to the
 * non-integration with payment services like HelloCash and M-Birr.
 * Payment happens through "agents". After an agent has issued a payment request for a user's
 * company, then the user needs to verify the payment has been made to continue using the app.
 * This is particularly necessary for uses that don't use the sync feature as the payment verification
 * should always happen on every sync.
 */
func (s *SheketController) VerifyPayment(c context.Context, request *sp.VerifyPaymentRequest) (response *sp.VerifyPaymentResponse, err error) {
	defer trace("VerifyPayment")()

	user_info, err := GetUserWithCompanyPermission(request.CompanyAuth)
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err.Error())
	}
	company, err := Store.GetCompanyById(user_info.CompanyId)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	license, err := GenerateCompanyLicense(user_info.CompanyId, user_info.User.UserId,
		company.EncodedPayment, request.DeviceId, request.LocalUserTime)

	if err != nil {
		return nil, grpc.Errorf(codes.DeadlineExceeded, "License expired, please renew, '%v'", err)
	}

	return &sp.VerifyPaymentResponse{SignedLicense: license}, nil
}

/**
 * Generates a signed license if there is still paid period left. This doesn't query the db, only uses
 * the info provided.
 */
func GenerateCompanyLicense(company_id, user_id int, encoded_payment, device_id, user_local_time string) (string, error) {
	payment_info, err := models.DecodePayment(encoded_payment)
	if err != nil {
		return "", err
	}

	current_date := time.Now().Unix()
	end_date := time.Unix(payment_info.IssuedDate, 0).
		AddDate(0, 0, int(payment_info.DurationInDays)).Unix()

	var remaining_days int

	payment_expired := false

	if current_date > end_date {
		payment_expired = true
	} else {
		remaining_days = int(
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
