package controller

import (
	"sheket/server/models"
	"net/http"
	"strconv"
)

const (
	INVALID_COMPANY_ID int64 = -1
	KEY_COMPANY_ID = "company_id"
)

var Store models.ShStore

func GetCurrentCompanyId(r *http.Request) int64 {
	id, err := strconv.ParseInt(r.Header.Get(KEY_COMPANY_ID), 10, 64)
	if err != nil {
		return INVALID_COMPANY_ID
	}
	return id
}