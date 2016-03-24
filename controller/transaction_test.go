package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sheket/server/models"
	"strings"
	"testing"
)

const (
	test_user_id         = 100
	test_company_id      = 23
	test_transaction_rev = 102
	test_item_rev        = 10213
	test_branch_item_rev = 121
)

func createTestTransaction(trans_id, local_id, branch_id,
	date int64, num_items int64) map[string]interface{} {
	trans := make(map[string]interface{})
	trans["trans_id"] = trans_id
	trans["local_id"] = local_id
	trans["branch_id"] = branch_id
	trans["date"] = date

	var items []interface{}
	if num_items > 0 {
		items = make([]interface{}, num_items)
		for i := int64(0); i < num_items; i++ {
			item := make([]interface{}, 4)
			item[0] = rand.Int63n(10)
			item[1] = rand.Int63n(100)
			item[2] = rand.Int63n(100)
			//item[3] = rand.Float64()
			item[3] = rand.Int63n(10000)

			items[i] = item
		}
	}
	trans["items"] = items
	return trans
}

var transactionSyncFormat string = `
	{
		"transaction_rev":%d,
		"branch_item_rev":%d,

		"transactions": %v
	}
`

var parseTransactionTests = []struct {
	trans_id  int64
	local_id  int64
	branch_id int64
	date      int64
	num_items int64
}{
	{-5, 100, 2, 1002, 1},
	{-6, 1027, 8, 201, 100},
}

func TestParseTransactionPost(t *testing.T) {
	trans_rev := int64(10)
	branch_item_rev := int64(100)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		transactions[i] = createTestTransaction(test.trans_id,
			test.local_id, test.branch_id, test.date, test.num_items)
	}

	trans_json, _ := json.MarshalIndent(transactions, "\t", "\t")
	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, string(trans_json))

	sync_data, err := parseTransactionPost(
		strings.NewReader(trans_test_string))

	if err != nil {
		t.Errorf("parsing error '%v'\n", err)
	} else if sync_data == nil {
		t.Errorf("nil parse data")
	}

	if sync_data.UserTransRev != trans_rev {
		t.Errorf("trans rev error, want %d, got %d", trans_rev,
			sync_data.UserTransRev)
	}
	if sync_data.UserBranchItemRev != branch_item_rev {
		t.Errorf("branch item rev error, want %d, got %d", branch_item_rev,
			sync_data.UserBranchItemRev)
	}

	if len(sync_data.NewTrans) != len(parseTransactionTests) {
		t.Errorf("num transactions error, wanted %d, got %d",
			len(parseTransactionTests), len(sync_data.NewTrans))
	}

	for i, trans := range sync_data.NewTrans {
		got := trans
		expected := parseTransactionTests[i]
		if got.TransactionId != expected.trans_id {
			t.Errorf("got:%d, expected_id wanted %d, expected %d",
				i, expected.trans_id, got.TransactionId)
		}
		if got.LocalTransactionId != expected.local_id {
			t.Errorf("got:%d, local_id wanted %d, expected %d",
				i, expected.local_id, got.LocalTransactionId)
		}
		if got.BranchId != expected.branch_id {
			t.Errorf("got:%d, branch_id wanted %d, expected %d",
				i, expected.branch_id, got.BranchId)
		}
		if got.Date != expected.date {
			t.Errorf("got:%d, date wanted %d, expected %d",
				i, expected.date, got.Date)
		}

		if int64(len(got.TransItems)) != expected.num_items {
			t.Errorf("got:%d, num_items wanted %d, expected %d",
				i, expected.num_items, len(got.TransItems))
		}
	}
}

func TestParseMissingTransArr(t *testing.T) {
	trans_rev := int64(10)
	branch_item_rev := int64(100)

	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, "")

	_, err := parseTransactionPost(
		strings.NewReader(trans_test_string))

	if err == nil {
		t.Errorf("expecting parse error")
	}
}

func TestParseEmptyTransArr(t *testing.T) {
	trans_rev := int64(10)
	branch_item_rev := int64(100)

	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, "[]")

	_, err := parseTransactionPost(
		strings.NewReader(trans_test_string))

	if err != nil {
		t.Errorf("parse error %v", err)
	}
}

func TestParseMissingTransId(t *testing.T) {
	trans_rev := int64(10)
	branch_item_rev := int64(100)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		trans := createTestTransaction(test.trans_id,
			test.local_id, test.branch_id, test.date, test.num_items)
		delete(trans, "trans_id")
		transactions[i] = trans
	}

	trans_json, _ := json.MarshalIndent(transactions, "\t", "\t")
	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, string(trans_json))

	_, err := parseTransactionPost(
		strings.NewReader(trans_test_string))

	if err != nil {
		t.Errorf("parsing error '%v'\n", err)
	}
}

func TestParseInvalidItems(t *testing.T) {
	trans_rev := int64(10)
	branch_item_rev := int64(100)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		trans := createTestTransaction(test.trans_id,
			test.local_id, test.branch_id, test.date, test.num_items)
		delete(trans, "items")

		items := []interface{}{
			strings.Split("a b c d", " "),
			strings.Split("4 2 c d", " "),
			strings.Split("2 2 2 1", " "),
		}

		trans["items"] = items
		transactions[i] = trans
	}

	trans_json, _ := json.MarshalIndent(transactions, "\t", "\t")
	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, string(trans_json))

	_, err := parseTransactionPost(
		strings.NewReader(trans_test_string))

	if err == nil {
		t.Errorf("expected number parsing error")
	}
}

type t_trans_items struct {
	trans_type      int64
	item_id         int64
	other_branch_id int64
	quantity        float64
	existInBranch   bool
}

type t_branch_item_qty struct {
	branch_id     int64
	item_id       int64
	quantity_left float64
}

var (
	t_branch_1 int64 = 1
	t_branch_2 int64 = 2
	t_branch_3 int64 = 3

	t_item_1 int64 = 11
	t_item_2 int64 = 12
	t_item_3 int64 = 13

	t_company_id int64 = 10

	initialQty = []struct {
		branch_id, item_id int64
		initial_qty        float64
	}{
		// if not listed here, 0 is the default initial quantity
		{t_branch_1, t_item_1, 30},
		{t_branch_2, t_item_3, 20},
	}

	addTransactionsTests = []struct {
		created_trans_id int64

		branch_id   int64
		trans_items []t_trans_items
	}{
		{2, t_branch_1,
			[]t_trans_items{
				// the transaction items
				t_trans_items{models.TRANS_TYPE_ADD_PURCHASED_ITEM, t_item_1, -1, 10, true},
				t_trans_items{models.TRANS_TYPE_SELL_PURCHASED_ITEM_DIRECTLY, t_item_1, -1, 20, true},

				// sell item_2 from shop, qty_left 0 => -70
				t_trans_items{models.TRANS_TYPE_SELL_CURRENT_BRANCH_ITEM, t_item_2, -1, 70, false},

				// transfer item_1 to branch_1, branch_2 qty_left 0 => -100
				t_trans_items{models.TRANS_TYPE_TRANSFER_OTHER_BRANCH_ITEM, t_item_1, t_branch_2, 100, false},

				t_trans_items{models.TRANS_TYPE_SELL_CURRENT_BRANCH_ITEM, t_item_1, -1, 70, true},
			},
		},
		{3, t_branch_2,
			[]t_trans_items{
				// the transaction items
				t_trans_items{models.TRANS_TYPE_SELL_PURCHASED_ITEM_DIRECTLY, t_item_3, -1, 1000, true},
				t_trans_items{models.TRANS_TYPE_TRANSFER_OTHER_BRANCH_ITEM, t_item_1, t_branch_3, 30, false},

				// sell item_2 from shop, qty_left 0 => -70
				t_trans_items{models.TRANS_TYPE_SELL_CURRENT_BRANCH_ITEM, t_item_3, -1, 70, true},

				t_trans_items{models.TRANS_TYPE_ADD_PURCHASED_ITEM, t_item_1, -1, 700, true},
			},
		},
	}

	branch_item_qty = []t_branch_item_qty{
		t_branch_item_qty{t_branch_1, t_item_1, 70},
		t_branch_item_qty{t_branch_1, t_item_2, -70},

		t_branch_item_qty{t_branch_2, t_item_1, 630},
		t_branch_item_qty{t_branch_2, t_item_3, -50},

		t_branch_item_qty{t_branch_3, t_item_1, -30},
	}
)

func TestAddTransactionSimpleMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	save_store := Store
	defer func() {
		Store = save_store
	}()

	mock := models.NewComposableShStoreMock(ctrl)
	Store = mock
	start_qty := make(map[models.BranchItemPair]float64, 10)
	for _, item := range initialQty {
		start_qty[models.BranchItemPair{item.branch_id, item.item_id}] = item.initial_qty
	}
	mock.BranchItemStore = models.NewSimpleBranchItemStore(start_qty)
	mock.TransactionStore = models.NewSimpleTransactionStore()

	transactions := make([]*models.ShTransaction, len(addTransactionsTests))
	for i, test := range addTransactionsTests {
		transactions[i] = &models.ShTransaction{}
		transactions[i].BranchId = test.branch_id

		transactions[i].TransItems = make([]*models.ShTransactionItem, len(test.trans_items))
		for j, item := range test.trans_items {
			transactions[i].TransItems[j] = &models.ShTransactionItem{TransType: item.trans_type,
				ItemId: item.item_id, OtherBranchId: item.other_branch_id,
				Quantity: item.quantity}
		}
	}

	tnx := &sql.Tx{}
	trans_result, err := addTransactionsToDataStore(tnx, transactions, t_company_id)
	if err != nil {
		t.Fatalf("add transactions error '%v'", err)
	}

	for _, item_qty := range branch_item_qty {
		branch_id := item_qty.branch_id
		item_id := item_qty.item_id
		qty_left := item_qty.quantity_left

		branch_item, ok := trans_result.AffectedBranchItems[Pair_BranchItem{branch_id, item_id}]
		if !ok {
			t.Errorf("item %d in branch %d missing\n", item_id, branch_id)
			continue
		}

		float64_equal := func(a, b float64) bool {
			return math.Abs(a-b) < 0.0001
		}

		if !float64_equal(branch_item.Quantity, qty_left) {
			t.Errorf("item %d in branch %d, quantiy error. wanted %f got %f\n",
				item_id, branch_id, qty_left, branch_item.Quantity)
		}
	}
}

func setUpTransactions(mock *models.MockShStore, tnx *sql.Tx) []*models.ShTransaction {
	seenItems := make(map[Pair_BranchItem]bool, 10)

	var transactions []*models.ShTransaction
	for _, test := range addTransactionsTests {
		trans := &models.ShTransaction{}
		trans.BranchId = test.branch_id

		trans.TransItems = make([]*models.ShTransactionItem, len(test.trans_items))
		for i, item := range test.trans_items {
			trans.TransItems[i] = &models.ShTransactionItem{TransType: item.trans_type,
				ItemId: item.item_id, OtherBranchId: item.other_branch_id,
				Quantity: item.quantity}

			expect_get_item := func(branch_id, item_id int64, exist bool) {

				if seenItems[Pair_BranchItem{branch_id, item_id}] {
					return
				}
				seenItems[Pair_BranchItem{branch_id, item_id}] = true

				get_initial_qty := func() float64 {
					for _, itemQty := range initialQty {
						if itemQty.branch_id == branch_id &&
							itemQty.item_id == item_id {
							return itemQty.initial_qty
						}
					}
					return float64(0)
				}

				call := mock.EXPECT().GetBranchItemInTx(tnx,
					branch_id, item_id)
				if exist {
					call.Return(&models.ShBranchItem{
						CompanyId: t_company_id, BranchId: branch_id,
						ItemId:   item_id,
						Quantity: get_initial_qty(),
					}, nil)
				} else {
					call.Return(nil,
						fmt.Errorf("test error, item doesn't exist in branch"))
				}
			}

			expect_get_item(trans.BranchId, item.item_id, item.existInBranch)
			if item.trans_type == models.TRANS_TYPE_TRANSFER_OTHER_BRANCH_ITEM ||
				item.trans_type == models.TRANS_TYPE_SELL_OTHER_BRANCH_ITEM {

				expect_get_item(item.other_branch_id, item.item_id, item.existInBranch)
			}
		}

		transactions = append(transactions, trans)

		created := *trans
		created.TransactionId = test.created_trans_id
		mock.EXPECT().CreateShTransaction(tnx, trans).Return(&created, nil)
	}
	return transactions
}

func TestAddTransactionFinalQuantity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	save_store := Store
	defer func() {
		Store = save_store
	}()

	mock := models.NewMockShStore(ctrl)
	Store = mock

	tnx := &sql.Tx{}
	transactions := setUpTransactions(mock, tnx)

	trans_result, err := addTransactionsToDataStore(tnx, transactions, t_company_id)
	if err != nil {
		t.Fatalf("add transactions error '%v'", err)
	}

	for _, item_qty := range branch_item_qty {
		branch_id := item_qty.branch_id
		item_id := item_qty.item_id
		qty_left := item_qty.quantity_left

		branch_item, ok := trans_result.AffectedBranchItems[Pair_BranchItem{branch_id, item_id}]
		if !ok {
			t.Errorf("item %d in branch %d missing\n", item_id, branch_id)
			continue
		}

		float64_equal := func(a, b float64) bool {
			return math.Abs(a-b) < 0.0001
		}

		if !float64_equal(branch_item.Quantity, qty_left) {
			t.Errorf("item %d in branch %d, quantiy error. wanted %f got %f\n",
				item_id, branch_id, qty_left, branch_item.Quantity)
		}
	}
}

const (
	trans_rev       = int64(10)
	branch_item_rev = int64(100)
	company_id      = int64(100)
	user_id         = int64(12)
)


var ctrl *gomock.Controller
var mock *models.ComposableShStoreMock
var save_store models.ShStore
var user *models.User

var save_getter func(*http.Request)(*models.User, error)

func setup_user(t *testing.T, user_id int64) {
	save_getter = currentUserGetter
	user = &models.User{UserId: user_id}
	currentUserGetter = func(*http.Request) (*models.User, error) {
		return user, nil
	}
}

func teardown_user() {
	currentUserGetter = save_getter
}

func setup_store(t *testing.T) {
	save_store = Store
	ctrl = gomock.NewController(t)
	mock = models.NewComposableShStoreMock(ctrl)
	Store = mock
}

func teardown_store() {
	ctrl.Finish()
	Store = save_store
}

var tnx_setup bool = false
var tnx *sql.Tx
var db *sql.DB
var db_mock sqlmock.Sqlmock

func setup_tnx(t *testing.T) {
	tnx_setup = true
	var err error
	db, db_mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when testing db", err)
	}

	db_mock.ExpectBegin()
	tnx, _ = db.Begin()
}

func teardown_tnx() {
	if tnx_setup {
		db.Close()
	}
	tnx_setup = false
}

func TestTransactionHandler(t *testing.T) {
	setup_store(t)
	defer teardown_store()

	setup_user(t, user_id)
	defer teardown_user()

	setup_tnx(t)
	defer teardown_tnx()

	start_qty := make(map[models.BranchItemPair]float64, 10)
	for _, item := range initialQty {
		start_qty[models.BranchItemPair{item.branch_id, item.item_id}] = item.initial_qty
	}
	mock.BranchItemStore = models.NewSimpleBranchItemStore(start_qty)
	mock.TransactionStore = models.NewSimpleTransactionStore()

	source := models.NewMockSource(ctrl)
	source.EXPECT().Begin().Return(tnx, nil)
	mock.Source = source

	user_store := models.NewMockUserStore(ctrl)
	permission := &models.UserPermission{CompanyId: company_id,
		UserId: user_id, PermissionType: models.U_PERMISSION_MANAGER, BranchId: -1}
	user_store.EXPECT().GetUserPermission(user, company_id).Return(permission, nil)
	mock.UserStore = user_store

	mock.RevisionStore = models.NewSimpleRevisionStore(nil)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		transactions[i] = createTestTransaction(test.trans_id,
			test.local_id, test.branch_id, test.date, test.num_items)
	}

	str, err := json.MarshalIndent(transactions, "\t", "\t")
	if err != nil {
		t.Fatalf("malformed transactions %v", err)
	}
	trans_post := fmt.Sprintf(transactionSyncFormat, trans_rev, branch_item_rev, string(str))

	req, err := http.NewRequest("POST", "www.example.com", strings.NewReader(trans_post))
	if err != nil {
		t.Fatalf("request error '%v'", err)
	}
	req.Header.Set(KEY_COMPANY_ID, fmt.Sprintf("%d", company_id))

	w := httptest.NewRecorder()
	TransactionSyncHandler(w, req)
	if w.Code != http.StatusOK {
		t.Logf("Handler exited with non ok status code %s",
			http.StatusText(w.Code))
	}
	//t.Logf("%s\n", w.Body.Bytes())
	//t.Logf("Size :%d", len(w.Body.Bytes()))
}

/**
 * Benchmark tests
 */
var parseTransactionBenchs = []struct {
	trans_id  int64
	local_id  int64
	branch_id int64
	date      int64
	num_items int64
}{{-5, 100, 2, 1002, 300},
	{-6, 1027, 8, 201, 0},
	{-6, 1027, 8, 201, 100},
}

func BenchmarkParseTransactionPost(b *testing.B) {
	trans_rev := int64(10)
	branch_item_rev := int64(100)

	transactions := make([]interface{}, len(parseTransactionBenchs))
	for i, test := range parseTransactionBenchs {
		transactions[i] = createTestTransaction(test.trans_id,
			test.local_id, test.branch_id, test.date, test.num_items)
	}

	trans_json, _ := json.MarshalIndent(transactions, "\t", "\t")
	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, string(trans_json))

	b.Logf("Size of test in bytes: %d\n", len(trans_test_string))
	for i := 0; i < b.N; i++ {
		_, err := parseTransactionPost(
			strings.NewReader(trans_test_string))
		if err != nil {
			b.Errorf("%d parse failed\n", i)
		}
	}
}
