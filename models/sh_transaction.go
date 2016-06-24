package models

import (
	"database/sql"
	"fmt"
)

type ShTransaction struct {
	CompanyId     int64
	TransactionId int64
	ClientUUID    string

	UserId     int64
	BranchId   int64
	Date       int64
	TransItems []*ShTransactionItem
}

const (
	TRANS_JSON_TRANS_ID  = "trans_id"
	TRANS_JSON_UUID      = "client_uuid"
	TRANS_JSON_USER_ID   = "user_id"
	TRANS_JSON_BRANCH_ID = "branch_id"
	TRANS_JSON_DATE      = "date"
	TRANS_JSON_ITEMS     = "items"
)

type ShTransactionItem struct {
	CompanyId       int64
	TransactionId   int64
	TransType       int64
	ItemId          int64
	OtherBranchId   int64
	Quantity        float64
	TransactionNote string
}

const (
	// Transaction Type constants

	TRANS_TYPE_ADD_PURCHASED           int64 = 1 // Increase stock count from purchased merchandise
	TRANS_TYPE_ADD_RETURN_ITEM         int64 = 2 // When a customer returns an item
	TRANS_TYPE_ADD_TRANSFER_FROM_OTHER int64 = 3 // Increase stock from transfer from another branch

	TRANS_TYPE_SUB_CURRENT_BRANCH_SALE int64 = 11 // Decrease stock by selling current branch inventory
	TRANS_TYPE_SUB_TRANSFER_TO_OTHER   int64 = 12 // Decrease stock by sending inventory to other branch
)

func (s *shStore) CreateShTransactionInTx(tnx *sql.Tx, trans *ShTransaction) (*ShTransaction, error) {
	err := tnx.QueryRow(
		fmt.Sprintf("insert into %s "+
			"(company_id, user_id, branch_id, t_date, client_uuid) values "+
			"($1, $2, $3, $4, $5) RETURNING transaction_id",
			TABLE_TRANSACTION),
		trans.CompanyId, trans.UserId, trans.BranchId, trans.Date, trans.ClientUUID).
		Scan(&trans.TransactionId)
	if err != nil {
		return nil, err
	}

	for i := range trans.TransItems {
		elem, err := s.AddShTransactionItemInTx(tnx, trans, trans.TransItems[i])
		if err != nil {
			return nil, err
		}
		trans.TransItems[i] = elem
	}

	return trans, nil
}

func (s *shStore) AddShTransactionItemInTx(tnx *sql.Tx, trans *ShTransaction, elem *ShTransactionItem) (*ShTransactionItem, error) {
	_, err := tnx.Exec(fmt.Sprintf("insert into %s "+
		"(company_id, transaction_id, trans_type, item_id, other_branch_id, quantity, trans_note) values "+
		"($1, $2, $3, $4, $5, $6, $7)", TABLE_TRANSACTION_ITEM),
		trans.CompanyId, trans.TransactionId, elem.TransType, elem.ItemId, elem.OtherBranchId, elem.Quantity, elem.TransactionNote)
	if err != nil {
		return nil, err
	}
	elem.TransactionId = trans.TransactionId
	return elem, nil
}

func (s *shStore) GetShTransactionById(company_id, trans_id int64, fetch_items bool) (*ShTransaction, error) {
	msg := fmt.Sprintf("company:%d, no transaction with id %d", company_id, trans_id)
	transaction, err := _queryShTransactions(s, fetch_items, msg,
		"where company_id = $1 AND transaction_id = $2", company_id, trans_id)
	if err != nil {
		return nil, err
	}
	if len(transaction) == 0 {
		return nil, fmt.Errorf("error fetching transaction:%d", trans_id)
	}
	return transaction[0], nil
}

func (s *shStore) GetShTransactionByUUIDInTx(tnx *sql.Tx, uid string) (*ShTransaction, error) {
	msg := fmt.Sprintf("no transaction with that uuid:%s", uid)
	transaction, err := _queryShTransactionsInTx(tnx, msg, "where client_uuid = $1", uid)
	if err != nil {
		return nil, err
	}
	if len(transaction) == 0 {
		return nil, nil
	}
	return transaction[0], nil
}

func (s *shStore) GetShTransactionSinceTransId(company_id, prev_id int64) (trans []*ShTransaction, err error) {
	msg := fmt.Sprintf("no transactions after id:%d", prev_id)
	transaction, err := _queryShTransactions(s, true, msg,
		"where company_id = $1 AND transaction_id > $2", company_id, prev_id)
	if err != nil {
		return nil, err
	}
	return transaction, nil
}

func _queryShTransactions(s *shStore, fetch_items bool, err_msg string, where_stmt string, args ...interface{}) ([]*ShTransaction, error) {
	var result []*ShTransaction

	query := fmt.Sprintf("select transaction_id, company_id, "+
		"branch_id, user_id, t_date, client_uuid from %s", TABLE_TRANSACTION)
	sort_by := " ORDER BY transaction_id asc"

	var rows *sql.Rows
	var err error
	if len(where_stmt) > 0 {
		rows, err = s.Query(query+" "+where_stmt+sort_by, args...)
	} else {
		rows, err = s.Query(query + sort_by)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v", err_msg, err)
	}

	defer rows.Close()

	for rows.Next() {
		t := new(ShTransaction)
		err := rows.Scan(
			&t.TransactionId,
			&t.CompanyId,
			&t.BranchId,
			&t.UserId,
			&t.Date,
			&t.ClientUUID,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		var items []*ShTransactionItem
		if fetch_items {
			items, err = _queryShTransactionItems(s, err_msg, "where transaction_id = $1", t.TransactionId)
			if err != nil {
				return nil, err
			}
		}
		t.TransItems = items
		result = append(result, t)
	}
	return result, nil
}

func _queryShTransactionsInTx(tnx *sql.Tx, err_msg string, where_stmt string, args ...interface{}) ([]*ShTransaction, error) {
	var result []*ShTransaction

	query := fmt.Sprintf("select transaction_id, company_id, "+
		"branch_id, user_id, t_date, client_uuid from %s", TABLE_TRANSACTION)
	sort_by := " ORDER BY transaction_id asc"

	var rows *sql.Rows
	var err error
	if len(where_stmt) > 0 {
		rows, err = tnx.Query(query+" "+where_stmt+sort_by, args...)
	} else {
		rows, err = tnx.Query(query + sort_by)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v", err_msg, err)
	}

	defer rows.Close()

	for rows.Next() {
		t := new(ShTransaction)
		err := rows.Scan(
			&t.TransactionId,
			&t.CompanyId,
			&t.BranchId,
			&t.UserId,
			&t.Date,
			&t.ClientUUID,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}
		result = append(result, t)
	}
	return result, nil
}

func _queryShTransactionItems(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShTransactionItem, error) {
	var result []*ShTransactionItem

	query := fmt.Sprintf("select company_id, transaction_id, trans_type, item_id, "+
		"other_branch_id, quantity, trans_note from %s", TABLE_TRANSACTION_ITEM)

	var rows *sql.Rows
	var err error
	if len(where_stmt) > 0 {
		rows, err = s.Query(query+" "+where_stmt, args...)
	} else {
		rows, err = s.Query(query)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v", err_msg, err)
	}

	defer rows.Close()

	for rows.Next() {
		i := new(ShTransactionItem)
		err := rows.Scan(
			&i.CompanyId,
			&i.TransactionId,
			&i.TransType,
			&i.ItemId,
			&i.OtherBranchId,
			&i.Quantity,
			&i.TransactionNote,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, i)
	}
	return result, nil
}
