package models

import (
	"database/sql"
	"fmt"
)

type ShTransaction struct {
	CompanyId     int64
	TransactionId int64

	// This is helpful in preventing duplicate transaction posting
	LocalTransactionId int64
	UserId             int64
	BranchId           int64
	Date               int64
	TransItems         []*ShTransactionItem
}

const (
	TRANS_TYPE_GENERIC                    = 1
	TRANS_TYPE_SELL_CURRENT_BRANCH_ITEM   = 2
	TRANS_TYPE_SELL_PURCHASE_ITEM         = 3
	TRANS_TYPE_SELL_OTHER_BRANCH_ITEM     = 4
	TRANS_TYPE_TRANSFER_OTHER_BRANCH_ITEM = 5
	TRANS_TYPE_ADD_PURCHASED_ITEM         = 6
)

type ShTransactionItem struct {
	TransactionId int64
	TransType     int64
	ItemId        int64
	OtherBranchId int64
	Quantity      float64
}

func (s *ShTransactionItem) Map() map[string]interface{} {
	result := make(map[string]interface{})
	result["transaction_id"] = s.TransactionId
	result["trans_type"] = s.TransType
	result["item_id"] = s.ItemId
	result["other_branch"] = s.OtherBranchId
	result["quantity"] = s.Quantity
	return result
}

func (s *shStore) CreateShTransaction(tnx *sql.Tx, trans *ShTransaction) (*ShTransaction, error) {
	exist_trans, err := tnx.Query(
		fmt.Sprintf("select transaction_id from %s "+
			"where company_id = $1 and user_id = $2 "+
			"and local_transaction_id = $3", TABLE_TRANSACTION),
		trans.CompanyId, trans.UserId, trans.LocalTransactionId)
	if err != nil {
		return nil, err
	}
	if exist_trans.Next() { // the transaction already exists, avoid duplicate
		exist_trans.Close()
		return nil, fmt.Errorf("Transaction already exists company:%d user:%d trans_local:%d",
			trans.CompanyId, trans.UserId, trans.LocalTransactionId)
	}
	exist_trans.Close()

	// the transaction_id is not autoincrement because it is only unique
	// to a company not globally. So each company business transaction will
	// be prev company transaction max + 1
	max_trans_id := int64(0)
	prev_trans, err := tnx.Query(
		fmt.Sprintf("select max(transaction_id) from %s "+
			"where company_id = $1", TABLE_TRANSACTION),
		trans.CompanyId)
	if err == nil && prev_trans.Next() {
		prev_trans.Scan(&max_trans_id)
	}

	max_trans_id += 1
	trans.TransactionId = max_trans_id

	_, err = tnx.Exec(fmt.Sprintf("insert into %s "+
		"(transaction_id, company_id, user_id, local_transaction_id, branch_id, date) values " +
		"($1, $2, $3, $4, $5, $6)",
		TABLE_TRANSACTION),
		max_trans_id, trans.CompanyId, trans.UserId, trans.LocalTransactionId, trans.BranchId, trans.Date)
	if err != nil {
		return nil, err
	}

	for i := range trans.TransItems {
		elem, err := s.AddShTransactionElem(tnx, trans, trans.TransItems[i])
		if err != nil {
			return nil, err
		}
		trans.TransItems[i] = elem
	}

	return trans, nil
}

func (s *shStore) AddShTransactionElem(tnx *sql.Tx, trans *ShTransaction, elem *ShTransactionItem) (*ShTransactionItem, error) {
	_, err := tnx.Exec(fmt.Sprintf("insert into %s "+
		"(transaction_id, trans_type, item_id, other_branch_id, quantity) values "+
		"($1, $2, $3, $4, $5)", TABLE_TRANSACTION_ITEM),
		trans.TransactionId, elem.TransType, elem.ItemId, elem.OtherBranchId, elem.Quantity)
	if err != nil {
		return nil, err
	}
	elem.TransactionId = trans.TransactionId
	return elem, nil
}

func (s *shStore) GetShTransactionById(int64, bool) (*ShTransaction, error) {
	return nil, nil
}

func (s *shStore) ListShTransactionSinceTransId(int64) ([]*ShTransaction, error) {
	return nil, nil
}
