package controller

/*
import (
	"strings"
	"fmt"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

func createTestTransaction(trans_id, branch_id, date int,
	num_items int) map[string]interface{} {
	trans := make(map[string]interface{})
	trans["trans_id"] = trans_id
	trans["branch_id"] = branch_id
	trans["date"] = date

	var items []interface{}
	if num_items > 0 {
		items = make([]interface{}, num_items)
		for i := int(0); i < num_items; i++ {
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
	trans_id  int
	branch_id int
	date      int
	num_items int
}{
	{-5, 2, 1002, 1},
	{-6, 8, 201, 10},
}

func TestParseTransactionPost(t *testing.T) {
	trans_rev := int(10)
	branch_item_rev := int(100)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		transactions[i] = createTestTransaction(test.trans_id,
			test.branch_id, test.date, test.num_items)
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
		if got.BranchId != expected.branch_id {
			t.Errorf("got:%d, branch_id wanted %d, expected %d",
				i, expected.branch_id, got.BranchId)
		}
		if got.Date != expected.date {
			t.Errorf("got:%d, date wanted %d, expected %d",
				i, expected.date, got.Date)
		}

		if int(len(got.TransItems)) != expected.num_items {
			t.Errorf("got:%d, num_items wanted %d, expected %d",
				i, expected.num_items, len(got.TransItems))
		}
	}
}

func TestParseMissingTransArr(t *testing.T) {
	trans_rev := int(10)
	branch_item_rev := int(100)

	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, "")

	_, err := parseTransactionPost(
		strings.NewReader(trans_test_string))

	if err == nil {
		t.Errorf("expecting parse error")
	}
}

func TestParseEmptyTransArr(t *testing.T) {
	trans_rev := int(10)
	branch_item_rev := int(100)

	trans_test_string := fmt.Sprintf(transactionSyncFormat,
		trans_rev, branch_item_rev, "[]")

	_, err := parseTransactionPost(
		strings.NewReader(trans_test_string))

	if err != nil {
		t.Errorf("parse error %v", err)
	}
}

func TestParseMissingTransId(t *testing.T) {
	trans_rev := int(10)
	branch_item_rev := int(100)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		trans := createTestTransaction(test.trans_id,
			test.branch_id, test.date, test.num_items)
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
	trans_rev := int(10)
	branch_item_rev := int(100)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		trans := createTestTransaction(test.trans_id,
			test.branch_id, test.date, test.num_items)
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
	trans_type      int
	item_id         int
	other_branch_id int
	quantity        float64
	existInBranch   bool
}

type t_branch_item_qty struct {
	branch_id     int
	item_id       int
	quantity_left float64
}

var (
	t_branch_1 int = 1
	t_branch_2 int = 2
	t_branch_3 int = 3

	t_item_1 int = 11
	t_item_2 int = 12
	t_item_3 int = 13

	initialQty = []struct {
		branch_id, item_id int
		initial_qty        float64
	}{
		// if not listed here, 0 is the default initial quantity
		{t_branch_1, t_item_1, 30},
		{t_branch_1, t_item_2, -60},
		{t_branch_2, t_item_3, 20},
	}

	addTransactionsTests = []struct {
		created_trans_id int

		branch_id   int
		trans_items []t_trans_items
	}{
		{2, t_branch_1,
			[]t_trans_items{
				// the transaction items
				t_trans_items{models.TRANS_TYPE_ADD_PURCHASED, t_item_1, -1, 10, true},
				t_trans_items{models.TRANS_TYPE_SUB_DIRECT_SALE, t_item_1, -1, 20, true},

				// sell item_2 from shop, qty_left 0 => -70
				t_trans_items{models.TRANS_TYPE_SUB_CURRENT_BRANCH_SALE, t_item_2, -1, 70, false},

				// transfer item_1 to branch_1, branch_2 qty_left 0 => -100
				t_trans_items{models.TRANS_TYPE_ADD_TRANSFER_FROM_OTHER, t_item_1, t_branch_2, 100, false},

				t_trans_items{models.TRANS_TYPE_SUB_CURRENT_BRANCH_SALE, t_item_1, -1, 70, true},
				t_trans_items{models.TRANS_TYPE_ADD_RETURN_ITEM, t_item_2, -1, 100, false},
			},
		},
		{3, t_branch_2,
			[]t_trans_items{
				// the transaction items
				t_trans_items{models.TRANS_TYPE_SUB_DIRECT_SALE, t_item_3, -1, 1000, true},
				t_trans_items{models.TRANS_TYPE_SUB_TRANSFER_TO_OTHER, t_item_1, t_branch_3, 30, false},

				// sell item_2 from shop, qty_left 0 => -70
				t_trans_items{models.TRANS_TYPE_SUB_CURRENT_BRANCH_SALE, t_item_3, -1, 70, true},

				t_trans_items{models.TRANS_TYPE_ADD_PURCHASED, t_item_1, -1, 700, true},
			},
		},
	}

	branch_item_qty = []t_branch_item_qty{
		t_branch_item_qty{t_branch_1, t_item_1, 70},
		t_branch_item_qty{t_branch_1, t_item_2, -30},

		t_branch_item_qty{t_branch_2, t_item_1, 570},
		t_branch_item_qty{t_branch_2, t_item_3, -50},

		t_branch_item_qty{t_branch_3, t_item_1, 30},
	}
)

func TestAddTransactionFinalQuantity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	save_store := Store
	defer func() {
		Store = save_store
	}()

	start_qty := make(map[models.BranchItemPair]float64, 10)
	for _, item := range initialQty {
		start_qty[models.BranchItemPair{item.branch_id, item.item_id}] = item.initial_qty
	}
	mock := models.NewComposableShStoreMock(ctrl)
	Store = mock
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


// This is just too complicated to be a unit-test
func setUpTransactionExpectation(mock *models.MockShStore, tnx *sql.Tx) []*models.ShTransaction {
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

			expect_get_item := func(branch_id, item_id int, exist bool) {

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
			if item.trans_type == models.TRANS_TYPE_ADD_TRANSFER_FROM_OTHER ||
				item.trans_type == models.TRANS_TYPE_SUB_TRANSFER_TO_OTHER {

				expect_get_item(item.other_branch_id, item.item_id, item.existInBranch)
			}
		}

		transactions = append(transactions, trans)

		created := *trans
		created.TransactionId = test.created_trans_id
		mock.EXPECT().CreateShTransactionInTx(tnx, trans).Return(&created, nil)
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
	transactions := setUpTransactionExpectation(mock, tnx)

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
	trans_rev       = int(10)
	branch_item_rev = int(100)
	company_id      = int(100)
	user_id         = int(12)
)

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
	t_mock.BranchItemStore = models.NewSimpleBranchItemStore(start_qty)
	t_mock.TransactionStore = models.NewSimpleTransactionStore()

	source := models.NewMockSource(t_ctrl)
	source.EXPECT().Begin().Return(t_tnx, nil)
	t_mock.Source = source

	user_store := models.NewMockUserStore(t_ctrl)
	permission := &models.UserPermission{PermissionType: models.PERMISSION_TYPE_MANAGER}
	permission.CompanyId = company_id
	permission.UserId = user_id
	permission.Encode()
	user_store.EXPECT().GetUserPermission(t_user, company_id).Return(permission, nil)
	t_mock.UserStore = user_store

	t_mock.RevisionStore = models.NewSimpleRevisionStore(nil)

	transactions := make([]interface{}, len(parseTransactionTests))
	for i, test := range parseTransactionTests {
		transactions[i] = createTestTransaction(test.trans_id,
			test.branch_id, test.date, test.num_items)
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
	req.Header.Set(JSON_KEY_COMPANY_ID, fmt.Sprintf("%d", company_id))

	w := httptest.NewRecorder()
	TransactionSyncHandler(w, req)
	if w.Code != http.StatusOK {
		t.Logf("Handler exited with non ok status code %s",
			http.StatusText(w.Code))
	}
}

var parseTransactionBenchTests = []struct {
	trans_id  int
	branch_id int
	date      int
	num_items int
}{{-5, 2, 1002, 300},
	{-6, 8, 201, 0},
	{-6, 8, 201, 100},
}

func BenchmarkParseTransactionPost(b *testing.B) {
	trans_rev := 10
	branch_item_rev := 100

	transactions := make([]interface{}, len(parseTransactionBenchTests))
	for i, test := range parseTransactionBenchTests {
		transactions[i] = createTestTransaction(test.trans_id,
			test.branch_id, test.date, test.num_items)
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
*/
