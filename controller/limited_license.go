package controller

import (
	"fmt"
	"sheket/server/controller/signature"
	"sheket/server/models"
)

func GenerateLimited30DayLicense() (string, error) {
	date_duration := 30

	// if we've reached here, it means the user has valid remaining payment
	contract := fmt.Sprintf(""+
		"duration:%d;"+
		"contract_type:%d",
		date_duration,
		models.PAYMENT_CONTRACT_UNLIMITED_ONE_TIME)

	signature, err := signature.SignBase64EncodeMessage(contract)
	if err != nil {
		return "", err
	}

	license := fmt.Sprintf("%s_||_%s", contract, signature)
	return license, nil
}
