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

type ShTransactionItem struct {
	TransactionId int64
	TransType     int64
	ItemId        int64
	OtherBranchId int64
	Quantity      float64
}

const (
	// Transaction Type constants

	// This affects stock by decreasing item in current branch
	TRANS_TYPE_SELL_CURRENT_BRANCH_ITEM = iota + 1

	// This DOES NOT stock as the items are sold directly
	// from purchase, so branch inventory isn't affected by it
	TRANS_TYPE_SELL_PURCHASED_ITEM_DIRECTLY

	// This is a 'delegate' type transaction where a purchase
	// happens in a branch but the items are taken out of the
	// inventory of another branch, so the 'seller' branch's
	// stock isn't affected, but it will decrease the branch
	// from which them items are taken out
	TRANS_TYPE_SELL_OTHER_BRANCH_ITEM

	// This is a 'resupply' type of transaction, where a shop
	// request items from another branch(e.g: a warehouse),
	// i.e: the shop's stock increases where by the warehouse's decrease
	TRANS_TYPE_TRANSFER_OTHER_BRANCH_ITEM

	// This is the primary means for inventory in a company to INCREASE
	// This increases stock count of any branch where the goods are
	// stored after purchase
	TRANS_TYPE_ADD_PURCHASED_ITEM
)

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
		"(transaction_id, company_id, user_id, local_transaction_id, branch_id, date) values "+
		"($1, $2, $3, $4, $5, $6)",
		TABLE_TRANSACTION),
		max_trans_id, trans.CompanyId, trans.UserId, trans.LocalTransactionId, trans.BranchId, trans.Date)
	if err != nil {
		return nil, err
	}

	for i := range trans.TransItems {
		elem, err := s.AddShTransactionItem(tnx, trans, trans.TransItems[i])
		if err != nil {
			return nil, err
		}
		trans.TransItems[i] = elem
	}

	return trans, nil
}

func (s *shStore) AddShTransactionItem(tnx *sql.Tx, trans *ShTransaction, elem *ShTransactionItem) (*ShTransactionItem, error) {
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

func (s *shStore) GetShTransactionById(id int64, fetch_items bool) (*ShTransaction, error) {
	msg := fmt.Sprintf("no transaction with id %d", id)
	_, transaction, err := _queryShTransaction(0, s, fetch_items, msg, "where transaction_id = $1", id)
	if err != nil {
		return nil, err
	}
	if len(transaction) == 0 {
		return nil, fmt.Errorf("No transaction with id:%d", id)
	}
	return transaction[0], nil
}

func (s *shStore) GetShTransactionSinceTransId(prev_id int64) (max_id int64,
	trans []*ShTransaction, err error) {

	msg := fmt.Sprintf("no transactions after id:%d", prev_id)
	max_id, transaction, err := _queryShTransaction(max_id, s, false, msg, "where transaction_id > $1", prev_id)
	if err != nil {
		return max_id, nil, err
	}
	return max_id, transaction, nil
}

func _queryShTransaction(prev_max_id int64, s *shStore, fetch_items bool, err_msg string, where_stmt string, args ...interface{}) (max_id int64,
	trans []*ShTransaction, err error) {
	var result []*ShTransaction

	max_id = prev_max_id
	query := fmt.Sprintf("select transaction_id, company_id, "+
		"local_transaction_id, user_id, date from %s", TABLE_TRANSACTION)
	sort_by := " ORDER BY transaction_id asc"

	var rows *sql.Rows
	if len(where_stmt) > 0 {
		rows, err = s.Query(query+" "+where_stmt+sort_by, args...)
	} else {
		rows, err = s.Query(query + sort_by)
	}
	if err != nil {
		return max_id, nil, fmt.Errorf("%s %v", err_msg, err)
	}

	for rows.Next() {
		t := new(ShTransaction)
		err := rows.Scan(
			&t.TransactionId,
			&t.CompanyId,
			&t.LocalTransactionId,
			&t.UserId,
			&t.Date,
		)
		if err != nil {
			return max_id, nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		if t.TransactionId > max_id {
			max_id = t.TransactionId
		}

		var items []*ShTransactionItem
		if fetch_items {
			items, err = _queryShTransactionItems(s, err_msg, "where transaction_id = $1", t.TransactionId)
			if err != nil {
				return max_id, nil, err
			}
		}
		t.TransItems = items
		result = append(result, t)
	}
	return max_id, result, nil
}

func _queryShTransactionItems(s *shStore, err_msg string, where_stmt string, args ...interface{}) ([]*ShTransactionItem, error) {
	var result []*ShTransactionItem

	query := fmt.Sprintf("select transaction_id, trans_type, item_id, "+
		"other_branch_id, quantity from %s", TABLE_TRANSACTION_ITEM)

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

	for rows.Next() {
		i := new(ShTransactionItem)
		err := rows.Scan(
			&i.TransactionId,
			&i.TransType,
			&i.ItemId,
			&i.OtherBranchId,
			&i.Quantity,
		)
		if err != nil {
			return nil, fmt.Errorf("%s %v", err_msg, err.Error())
		}

		result = append(result, i)
	}
	return result, nil
}
