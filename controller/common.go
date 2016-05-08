package controller

import (
	"net/http"
	"sheket/server/models"
	"strconv"
)

const (
	INVALID_COMPANY_ID int64 = -1

	JSON_KEY_COMPANY_ID = "company_id"

	KEY_JSON_ID_OLD = "o"
	KEY_JSON_ID_NEW = "n"
)

var Store models.ShStore

// Useful in map's as a key
// Without this, the key should be a 2-level thing
// e.g: map[outer_key]map[inner_key] object
type Pair_BranchItem struct {
	BranchId int64
	ItemId   int64
}

func GetCurrentCompanyId(r *http.Request) int64 {
	id, err := strconv.ParseInt(r.Header.Get(JSON_KEY_COMPANY_ID), 10, 64)
	if err != nil {
		return INVALID_COMPANY_ID
	}
	return id
}
