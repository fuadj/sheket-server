package models

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

const (
	transaction_id       = 100
	local_transaction_id = 1909
	trans_type           = TRANS_TYPE_SELL_OTHER_BRANCH_ITEM
	other_branch         = 5
	trans_quantity       = 4.0
	num_trans_items      = 4
)

func _dummyShTransactionItem(i int64) *ShTransactionItem {
	return &ShTransactionItem{
		TransType:     trans_type,
		ItemId:        item_id + i,
		OtherBranchId: other_branch,
		Quantity:      trans_quantity,
	}
}

func _dummyShTransaction() *ShTransaction {
	trans := &ShTransaction{CompanyId: company_id,
		LocalTransactionId: local_transaction_id,
		UserId:             user_id, BranchId: branch_id, Date: date,
		TransItems: make([]*ShTransactionItem, 0)}

	trans.TransItems = make([]*ShTransactionItem, num_trans_items)
	for i := 0; i < num_trans_items; i++ {
		trans.TransItems[i] = _dummyShTransactionItem(int64(i))
	}

	return trans
}

func _transItemInsertExpectation(i int64, return_error bool) *sqlmock.ExpectedExec {
	expect := mock.ExpectExec(
		fmt.Sprintf("insert into %s", TABLE_TRANSACTION_ITEM)).
		WithArgs(transaction_id, trans_type, item_id+i,
		other_branch, trans_quantity)
	if return_error {
		expect.WillReturnError(fmt.Errorf("insert error"))
	} else {
		expect.WillReturnResult(sqlmock.NewResult(1, i))
	}
	return expect
}

func _transPrevExistExpectation(prev_exist bool, return_error bool) *sqlmock.ExpectedQuery {
	rs := sqlmock.NewRows(_cols("transaction_id"))
	if prev_exist {
		rs.AddRow(transaction_id)
	}
	expect := mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(company_id, user_id, local_transaction_id)
	if return_error {
		expect.WillReturnError(fmt.Errorf("select error"))
	} else {
		expect.WillReturnRows(rs)
	}
	return expect
}

func _transMaxExpectation(return_error bool) *sqlmock.ExpectedQuery {
	expect := mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(company_id)
	if return_error {
		expect.WillReturnError(fmt.Errorf("select error"))
	} else {
		expect.WillReturnRows(sqlmock.NewRows(_cols("transaction_id")).
		// the -1 is so the next id will be the test's transaction_id
		AddRow(transaction_id - 1))
	}
	return expect
}

func _transInsertExpectation(return_error bool) *sqlmock.ExpectedExec {
	expect := mock.ExpectExec(fmt.Sprintf("insert into %s", TABLE_TRANSACTION)).
		WithArgs(transaction_id, company_id, user_id, local_transaction_id, branch_id, date)
	if return_error {
		expect.WillReturnError(fmt.Errorf("insert error"))
	} else {
		expect.WillReturnResult(sqlmock.NewResult(1, 1))
	}
	return expect
}

func TestCreateShTransactionNew(t *testing.T) {
	mock_setup(t, "TestCreateShTransactionNew")
	defer mock_teardown()

	trans := _dummyShTransaction()

	mock.ExpectBegin()
	_transPrevExistExpectation(false, false)
	_transMaxExpectation(false)
	_transInsertExpectation(false)
	for i := 0; i < len(trans.TransItems); i++ {
		_transItemInsertExpectation(int64(i), false)
	}

	tnx, _ := db.Begin()

	updated, err := store.CreateShTransaction(tnx, trans)
	if err != nil {
		_log_err("CreateShTransaction error '%v'", err)
	} else if updated.TransactionId != transaction_id {
		_log_err("Not expected trans id want:%d got:%d", transaction_id, updated.TransactionId)
	}
}

func TestCreateShTransactionNewFail(t *testing.T) {
	mock_setup(t, "TestCreateShTransactionNewFail")
	defer mock_teardown()

	trans := _dummyShTransaction()

	mock.ExpectBegin()
	_transPrevExistExpectation(false, true)

	tnx, _ := db.Begin()

	_, err := store.CreateShTransaction(tnx, trans)
	if err == nil {
		_log_err("expected error")
	}
}

func TestCreateShTransactionNewInsertTransFail(t *testing.T) {
	mock_setup(t, "TestCreateShTransactionNewInsertTransFail")
	defer mock_teardown()

	mock.ExpectBegin()
	_transPrevExistExpectation(false, false)
	_transMaxExpectation(false)
	_transInsertExpectation(true)
	tnx, _ := db.Begin()

	_, err := store.CreateShTransaction(tnx, _dummyShTransaction())
	if err == nil {
		_log_err("expected insert error")
	}
}

func TestCreateShTransactionNewInsertItemsFail(t *testing.T) {
	mock_setup(t, "TestCreateShTransactionNewInsertItemsFail")
	defer mock_teardown()

	trans := _dummyShTransaction()

	mock.ExpectBegin()
	_transPrevExistExpectation(false, false)
	_transMaxExpectation(false)
	_transInsertExpectation(false)
	for i := 0; i < len(trans.TransItems); i++ {
		fail := true
		_transItemInsertExpectation(int64(i), fail)
		if fail {	// we can't add expectations that won't be meet
			break
		}
	}
	tnx, _ := db.Begin()

	_, err := store.CreateShTransaction(tnx, trans)
	if err == nil {
		_log_err("expected insert error")
	}
}

func TestCreateShTransactionExistError(t *testing.T) {
	mock_setup(t, "TestCreateShTransactionExistError")
	defer mock_teardown()

	mock.ExpectBegin()
	_transPrevExistExpectation(true, false)
	tnx, _ := db.Begin()

	_, err := store.CreateShTransaction(tnx, _dummyShTransaction())
	if err == nil {
		_log_err("expected transaction already exist error")
	}
}
