package models

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

const (
	transaction_id  = 100
	trans_type      = TRANS_TYPE_ADD_TRANSFER_FROM_OTHER
	other_branch    = 5
	trans_quantity  = 4.0
	num_trans_items = 4
)

func _dummyShTransactionItem(i int64) *ShTransactionItem {
	return &ShTransactionItem{
		TransType:     trans_type,
		ItemId:        t_item_id + i,
		OtherBranchId: other_branch,
		Quantity:      trans_quantity,
	}
}

func _dummyShTransaction() *ShTransaction {
	trans := &ShTransaction{CompanyId: t_company_id,
		UserId: t_user_id, BranchId: t_branch_id, Date: t_date,
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
		WithArgs(transaction_id, trans_type, t_item_id+i,
			other_branch, trans_quantity)
	if return_error {
		expect.WillReturnError(fmt.Errorf("insert error"))
	} else {
		expect.WillReturnResult(sqlmock.NewResult(1, i))
	}
	return expect
}

/*
func _transPrevExistExpectation(prev_exist bool, return_error bool) *sqlmock.ExpectedQuery {
	rs := sqlmock.NewRows(_cols("transaction_id"))
	if prev_exist {
		rs.AddRow(transaction_id)
	}
	expect := mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(t_company_id, t_user_id)
	if return_error {
		expect.WillReturnError(fmt.Errorf("select error"))
	} else {
		expect.WillReturnRows(rs)
	}
	return expect
}
*/

func _transMaxExpectation(return_error bool) *sqlmock.ExpectedQuery {
	expect := mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(t_company_id)
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
		WithArgs(transaction_id, t_company_id, t_user_id, t_branch_id, t_date)
	if return_error {
		expect.WillReturnError(fmt.Errorf("insert error"))
	} else {
		expect.WillReturnResult(sqlmock.NewResult(1, 1))
	}
	return expect
}

func TestCreateShTransactionNew(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	trans := _dummyShTransaction()

	mock.ExpectBegin()
	_transMaxExpectation(false)
	_transInsertExpectation(false)
	for i := 0; i < len(trans.TransItems); i++ {
		_transItemInsertExpectation(int64(i), false)
	}

	tnx, _ := db.Begin()

	updated, err := store.CreateShTransactionInTx(tnx, trans)
	if err != nil {
		t.Errorf("CreateShTransaction error '%v'", err)
	} else if updated.TransactionId != transaction_id {
		t.Errorf("Not expected trans id want:%d got:%d", transaction_id, updated.TransactionId)
	}
}

func TestCreateShTransactionNewFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	trans := _dummyShTransaction()

	mock.ExpectBegin()

	tnx, _ := db.Begin()

	_, err := store.CreateShTransactionInTx(tnx, trans)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestCreateShTransactionNewInsertTransFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	_transMaxExpectation(false)
	_transInsertExpectation(true)
	tnx, _ := db.Begin()

	_, err := store.CreateShTransactionInTx(tnx, _dummyShTransaction())
	if err == nil {
		t.Errorf("expected insert error")
	}
}

func TestCreateShTransactionNewInsertItemsFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	trans := _dummyShTransaction()

	mock.ExpectBegin()
	_transMaxExpectation(false)
	_transInsertExpectation(false)
	for i := 0; i < len(trans.TransItems); i++ {
		fail := true
		_transItemInsertExpectation(int64(i), fail)
		if fail { // we can't add expectations that won't be meet
			break
		}
	}
	tnx, _ := db.Begin()

	_, err := store.CreateShTransactionInTx(tnx, trans)
	if err == nil {
		t.Errorf("expected insert error")
	}
}

func TestCreateShTransactionExistError(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	tnx, _ := db.Begin()

	_, err := store.CreateShTransactionInTx(tnx, _dummyShTransaction())
	if err == nil {
		t.Errorf("expected transaction already exist error")
	}
}

func _transItemQueryExpectation(n int64, return_error bool) *sqlmock.ExpectedQuery {
	expect := mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION_ITEM)).
		WithArgs(transaction_id)
	if return_error {
		expect.WillReturnError(fmt.Errorf("insert error"))
	} else {
		rows := sqlmock.NewRows(
			_cols("transaction_id, trans_type, item_id, " +
				"other_branch_id,quantity"))
		for i := int64(0); i < n; i++ {
			rows.AddRow(transaction_id, trans_type, t_item_id+i,
				other_branch, trans_quantity)
		}
		expect.WillReturnRows(rows)
	}
	return expect
}

func _transQueryRows() sqlmock.Rows {
	return sqlmock.NewRows(
		_cols("transaction_id,company_id,"+
			"user_id, date")).
		AddRow(transaction_id, t_company_id,
			t_user_id, t_date)
}

func TestGetShTransactionByIdFetchItems(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(transaction_id).
		WillReturnRows(_transQueryRows())
	_transItemQueryExpectation(num_trans_items, false)

	transaction, err := store.GetShTransactionById(t_company_id, transaction_id, true)
	if err != nil {
		t.Errorf("GetShTransactionById error '%v'", err)
	} else if len(transaction.TransItems) != num_trans_items {
		t.Errorf("wanted %d transaction items, got %d",
			num_trans_items, len(transaction.TransItems))
	}
}

func TestGetShTransactionByIdNoTransactionError(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(transaction_id).
		// make the query succeed, but return no rows on the cursor
		WillReturnRows(sqlmock.NewRows(_cols("transaction_id,company_id," +
			"user_id, date")))

	_, err := store.GetShTransactionById(t_company_id, transaction_id, true)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGetShTransactionByIdNoItemsFetch(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(transaction_id).
		WillReturnRows(_transQueryRows())

	transaction, err := store.GetShTransactionById(t_company_id, transaction_id, false)
	if err != nil {
		t.Errorf("GetShTransactionById error '%v'", err)
	} else if len(transaction.TransItems) != 0 {
		t.Errorf("wanted %d transaction items, got %d",
			0, len(transaction.TransItems))
	}
}

func TestGetShTransactionByIdFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(transaction_id).
		WillReturnError(fmt.Errorf("select error"))

	_, err := store.GetShTransactionById(t_company_id, transaction_id, true)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGetShTransactionByIdFetchItemsFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(
		fmt.Sprintf("select (.+) from %s", TABLE_TRANSACTION)).
		WithArgs(transaction_id).
		WillReturnRows(_transQueryRows())
	_transItemQueryExpectation(0, true)

	_, err := store.GetShTransactionById(t_company_id, transaction_id, true)
	if err == nil {
		t.Errorf("expected error")
	}
}
