// Automatically generated by MockGen. DO NOT EDIT!
// Source: sh_store.go

package models

import (
	sql "database/sql"
	gomock "github.com/golang/mock/gomock"
)

// Mock of TransactionStore interface
type MockTransactionStore struct {
	ctrl     *gomock.Controller
	recorder *_MockTransactionStoreRecorder
}

// Recorder for MockTransactionStore (not exported)
type _MockTransactionStoreRecorder struct {
	mock *MockTransactionStore
}

func NewMockTransactionStore(ctrl *gomock.Controller) *MockTransactionStore {
	mock := &MockTransactionStore{ctrl: ctrl}
	mock.recorder = &_MockTransactionStoreRecorder{mock}
	return mock
}

func (_m *MockTransactionStore) EXPECT() *_MockTransactionStoreRecorder {
	return _m.recorder
}

func (_m *MockTransactionStore) CreateShTransaction(_param0 *sql.Tx, _param1 *ShTransaction) (*ShTransaction, error) {
	ret := _m.ctrl.Call(_m, "CreateShTransaction", _param0, _param1)
	ret0, _ := ret[0].(*ShTransaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockTransactionStoreRecorder) CreateShTransaction(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateShTransaction", arg0, arg1)
}

func (_m *MockTransactionStore) AddShTransactionItem(_param0 *sql.Tx, _param1 *ShTransaction, _param2 *ShTransactionItem) (*ShTransactionItem, error) {
	ret := _m.ctrl.Call(_m, "AddShTransactionItem", _param0, _param1, _param2)
	ret0, _ := ret[0].(*ShTransactionItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockTransactionStoreRecorder) AddShTransactionItem(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddShTransactionItem", arg0, arg1, arg2)
}

func (_m *MockTransactionStore) GetShTransactionById(id int64, fetch_items bool) (*ShTransaction, error) {
	ret := _m.ctrl.Call(_m, "GetShTransactionById", id, fetch_items)
	ret0, _ := ret[0].(*ShTransaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockTransactionStoreRecorder) GetShTransactionById(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetShTransactionById", arg0, arg1)
}

func (_m *MockTransactionStore) GetShTransactionSinceTransId(start_id int64) (int64, []*ShTransaction, error) {
	ret := _m.ctrl.Call(_m, "GetShTransactionSinceTransId", start_id)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].([]*ShTransaction)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (_mr *_MockTransactionStoreRecorder) GetShTransactionSinceTransId(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetShTransactionSinceTransId", arg0)
}

// Mock of ItemStore interface
type MockItemStore struct {
	ctrl     *gomock.Controller
	recorder *_MockItemStoreRecorder
}

// Recorder for MockItemStore (not exported)
type _MockItemStoreRecorder struct {
	mock *MockItemStore
}

func NewMockItemStore(ctrl *gomock.Controller) *MockItemStore {
	mock := &MockItemStore{ctrl: ctrl}
	mock.recorder = &_MockItemStoreRecorder{mock}
	return mock
}

func (_m *MockItemStore) EXPECT() *_MockItemStoreRecorder {
	return _m.recorder
}

func (_m *MockItemStore) CreateItem(_param0 *ShItem) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "CreateItem", _param0)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockItemStoreRecorder) CreateItem(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateItem", arg0)
}

func (_m *MockItemStore) CreateItemInTx(_param0 *sql.Tx, _param1 *ShItem) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "CreateItemInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockItemStoreRecorder) CreateItemInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateItemInTx", arg0, arg1)
}

func (_m *MockItemStore) UpdateItemInTx(_param0 *sql.Tx, _param1 *ShItem) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "UpdateItemInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockItemStoreRecorder) UpdateItemInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "UpdateItemInTx", arg0, arg1)
}

func (_m *MockItemStore) GetItemById(_param0 int64) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemById", _param0)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockItemStoreRecorder) GetItemById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemById", arg0)
}

func (_m *MockItemStore) GetItemByIdInTx(_param0 *sql.Tx, _param1 int64) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemByIdInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockItemStoreRecorder) GetItemByIdInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemByIdInTx", arg0, arg1)
}

func (_m *MockItemStore) GetAllCompanyItems(_param0 int64) ([]*ShItem, error) {
	ret := _m.ctrl.Call(_m, "GetAllCompanyItems", _param0)
	ret0, _ := ret[0].([]*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockItemStoreRecorder) GetAllCompanyItems(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetAllCompanyItems", arg0)
}

// Mock of BranchStore interface
type MockBranchStore struct {
	ctrl     *gomock.Controller
	recorder *_MockBranchStoreRecorder
}

// Recorder for MockBranchStore (not exported)
type _MockBranchStoreRecorder struct {
	mock *MockBranchStore
}

func NewMockBranchStore(ctrl *gomock.Controller) *MockBranchStore {
	mock := &MockBranchStore{ctrl: ctrl}
	mock.recorder = &_MockBranchStoreRecorder{mock}
	return mock
}

func (_m *MockBranchStore) EXPECT() *_MockBranchStoreRecorder {
	return _m.recorder
}

func (_m *MockBranchStore) CreateBranch(_param0 *ShBranch) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "CreateBranch", _param0)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchStoreRecorder) CreateBranch(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateBranch", arg0)
}

func (_m *MockBranchStore) CreateBranchInTx(_param0 *sql.Tx, _param1 *ShBranch) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "CreateBranchInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchStoreRecorder) CreateBranchInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateBranchInTx", arg0, arg1)
}

func (_m *MockBranchStore) UpdateBranchInTx(_param0 *sql.Tx, _param1 *ShBranch) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "UpdateBranchInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchStoreRecorder) UpdateBranchInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "UpdateBranchInTx", arg0, arg1)
}

func (_m *MockBranchStore) GetBranchById(_param0 int64) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "GetBranchById", _param0)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchStoreRecorder) GetBranchById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchById", arg0)
}

func (_m *MockBranchStore) GetBranchByIdInTx(_param0 *sql.Tx, _param1 int64) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "GetBranchByIdInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchStoreRecorder) GetBranchByIdInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchByIdInTx", arg0, arg1)
}

func (_m *MockBranchStore) ListCompanyBranches(_param0 int64) ([]*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "ListCompanyBranches", _param0)
	ret0, _ := ret[0].([]*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchStoreRecorder) ListCompanyBranches(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ListCompanyBranches", arg0)
}

// Mock of BranchItemStore interface
type MockBranchItemStore struct {
	ctrl     *gomock.Controller
	recorder *_MockBranchItemStoreRecorder
}

// Recorder for MockBranchItemStore (not exported)
type _MockBranchItemStoreRecorder struct {
	mock *MockBranchItemStore
}

func NewMockBranchItemStore(ctrl *gomock.Controller) *MockBranchItemStore {
	mock := &MockBranchItemStore{ctrl: ctrl}
	mock.recorder = &_MockBranchItemStoreRecorder{mock}
	return mock
}

func (_m *MockBranchItemStore) EXPECT() *_MockBranchItemStoreRecorder {
	return _m.recorder
}

func (_m *MockBranchItemStore) AddItemToBranch(_param0 *ShBranchItem) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "AddItemToBranch", _param0)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchItemStoreRecorder) AddItemToBranch(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddItemToBranch", arg0)
}

func (_m *MockBranchItemStore) AddItemToBranchInTx(_param0 *sql.Tx, _param1 *ShBranchItem) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "AddItemToBranchInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchItemStoreRecorder) AddItemToBranchInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddItemToBranchInTx", arg0, arg1)
}

func (_m *MockBranchItemStore) GetBranchItem(branch_id int64, item_id int64) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetBranchItem", branch_id, item_id)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchItemStoreRecorder) GetBranchItem(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchItem", arg0, arg1)
}

func (_m *MockBranchItemStore) GetBranchItemInTx(tnx *sql.Tx, branch_id int64, item_id int64) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetBranchItemInTx", tnx, branch_id, item_id)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchItemStoreRecorder) GetBranchItemInTx(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchItemInTx", arg0, arg1, arg2)
}

func (_m *MockBranchItemStore) UpdateBranchItemInTx(_param0 *sql.Tx, _param1 *ShBranchItem) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "UpdateBranchItemInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchItemStoreRecorder) UpdateBranchItemInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "UpdateBranchItemInTx", arg0, arg1)
}

func (_m *MockBranchItemStore) GetItemsInBranch(_param0 int64) ([]*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemsInBranch", _param0)
	ret0, _ := ret[0].([]*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchItemStoreRecorder) GetItemsInBranch(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemsInBranch", arg0)
}

func (_m *MockBranchItemStore) GetItemsInAllCompanyBranches(_param0 int64) ([]*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemsInAllCompanyBranches", _param0)
	ret0, _ := ret[0].([]*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockBranchItemStoreRecorder) GetItemsInAllCompanyBranches(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemsInAllCompanyBranches", arg0)
}

// Mock of CompanyStore interface
type MockCompanyStore struct {
	ctrl     *gomock.Controller
	recorder *_MockCompanyStoreRecorder
}

// Recorder for MockCompanyStore (not exported)
type _MockCompanyStoreRecorder struct {
	mock *MockCompanyStore
}

func NewMockCompanyStore(ctrl *gomock.Controller) *MockCompanyStore {
	mock := &MockCompanyStore{ctrl: ctrl}
	mock.recorder = &_MockCompanyStoreRecorder{mock}
	return mock
}

func (_m *MockCompanyStore) EXPECT() *_MockCompanyStoreRecorder {
	return _m.recorder
}

func (_m *MockCompanyStore) CreateCompany(u *User, c *Company) (*Company, error) {
	ret := _m.ctrl.Call(_m, "CreateCompany", u, c)
	ret0, _ := ret[0].(*Company)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockCompanyStoreRecorder) CreateCompany(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateCompany", arg0, arg1)
}

func (_m *MockCompanyStore) CreateCompanyInTx(_param0 *sql.Tx, _param1 *User, _param2 *Company) (*Company, error) {
	ret := _m.ctrl.Call(_m, "CreateCompanyInTx", _param0, _param1, _param2)
	ret0, _ := ret[0].(*Company)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockCompanyStoreRecorder) CreateCompanyInTx(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateCompanyInTx", arg0, arg1, arg2)
}

func (_m *MockCompanyStore) GetCompanyById(_param0 int64) (*Company, error) {
	ret := _m.ctrl.Call(_m, "GetCompanyById", _param0)
	ret0, _ := ret[0].(*Company)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockCompanyStoreRecorder) GetCompanyById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetCompanyById", arg0)
}

// Mock of UserStore interface
type MockUserStore struct {
	ctrl     *gomock.Controller
	recorder *_MockUserStoreRecorder
}

// Recorder for MockUserStore (not exported)
type _MockUserStoreRecorder struct {
	mock *MockUserStore
}

func NewMockUserStore(ctrl *gomock.Controller) *MockUserStore {
	mock := &MockUserStore{ctrl: ctrl}
	mock.recorder = &_MockUserStoreRecorder{mock}
	return mock
}

func (_m *MockUserStore) EXPECT() *_MockUserStoreRecorder {
	return _m.recorder
}

func (_m *MockUserStore) CreateUser(u *User) (*User, error) {
	ret := _m.ctrl.Call(_m, "CreateUser", u)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) CreateUser(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateUser", arg0)
}

func (_m *MockUserStore) CreateUserInTx(tnx *sql.Tx, u *User) (*User, error) {
	ret := _m.ctrl.Call(_m, "CreateUserInTx", tnx, u)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) CreateUserInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateUserInTx", arg0, arg1)
}

func (_m *MockUserStore) FindUserByName(_param0 string) (*User, error) {
	ret := _m.ctrl.Call(_m, "FindUserByName", _param0)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) FindUserByName(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindUserByName", arg0)
}

func (_m *MockUserStore) FindUserById(_param0 int64) (*User, error) {
	ret := _m.ctrl.Call(_m, "FindUserById", _param0)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) FindUserById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindUserById", arg0)
}

func (_m *MockUserStore) FindUserByNameInTx(_param0 *sql.Tx, _param1 string) (*User, error) {
	ret := _m.ctrl.Call(_m, "FindUserByNameInTx", _param0, _param1)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) FindUserByNameInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindUserByNameInTx", arg0, arg1)
}

func (_m *MockUserStore) SetUserPermission(_param0 *UserPermission) (*UserPermission, error) {
	ret := _m.ctrl.Call(_m, "SetUserPermission", _param0)
	ret0, _ := ret[0].(*UserPermission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) SetUserPermission(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetUserPermission", arg0)
}

func (_m *MockUserStore) SetUserPermissionInTx(_param0 *sql.Tx, _param1 *UserPermission) (*UserPermission, error) {
	ret := _m.ctrl.Call(_m, "SetUserPermissionInTx", _param0, _param1)
	ret0, _ := ret[0].(*UserPermission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) SetUserPermissionInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetUserPermissionInTx", arg0, arg1)
}

func (_m *MockUserStore) GetUserPermission(u *User, company_id int64) (*UserPermission, error) {
	ret := _m.ctrl.Call(_m, "GetUserPermission", u, company_id)
	ret0, _ := ret[0].(*UserPermission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUserStoreRecorder) GetUserPermission(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetUserPermission", arg0, arg1)
}

// Mock of RevisionStore interface
type MockRevisionStore struct {
	ctrl     *gomock.Controller
	recorder *_MockRevisionStoreRecorder
}

// Recorder for MockRevisionStore (not exported)
type _MockRevisionStoreRecorder struct {
	mock *MockRevisionStore
}

func NewMockRevisionStore(ctrl *gomock.Controller) *MockRevisionStore {
	mock := &MockRevisionStore{ctrl: ctrl}
	mock.recorder = &_MockRevisionStoreRecorder{mock}
	return mock
}

func (_m *MockRevisionStore) EXPECT() *_MockRevisionStoreRecorder {
	return _m.recorder
}

func (_m *MockRevisionStore) AddEntityRevisionInTx(_param0 *sql.Tx, _param1 *ShEntityRevision) (*ShEntityRevision, error) {
	ret := _m.ctrl.Call(_m, "AddEntityRevisionInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShEntityRevision)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockRevisionStoreRecorder) AddEntityRevisionInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddEntityRevisionInTx", arg0, arg1)
}

func (_m *MockRevisionStore) GetRevisionsSince(start_from *ShEntityRevision) (int64, []*ShEntityRevision, error) {
	ret := _m.ctrl.Call(_m, "GetRevisionsSince", start_from)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].([]*ShEntityRevision)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (_mr *_MockRevisionStoreRecorder) GetRevisionsSince(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetRevisionsSince", arg0)
}

// Mock of Source interface
type MockSource struct {
	ctrl     *gomock.Controller
	recorder *_MockSourceRecorder
}

// Recorder for MockSource (not exported)
type _MockSourceRecorder struct {
	mock *MockSource
}

func NewMockSource(ctrl *gomock.Controller) *MockSource {
	mock := &MockSource{ctrl: ctrl}
	mock.recorder = &_MockSourceRecorder{mock}
	return mock
}

func (_m *MockSource) EXPECT() *_MockSourceRecorder {
	return _m.recorder
}

func (_m *MockSource) GetDataStore() DataStore {
	ret := _m.ctrl.Call(_m, "GetDataStore")
	ret0, _ := ret[0].(DataStore)
	return ret0
}

func (_mr *_MockSourceRecorder) GetDataStore() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetDataStore")
}

func (_m *MockSource) Begin() (*sql.Tx, error) {
	ret := _m.ctrl.Call(_m, "Begin")
	ret0, _ := ret[0].(*sql.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSourceRecorder) Begin() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Begin")
}

// Mock of ShStore interface
type MockShStore struct {
	ctrl     *gomock.Controller
	recorder *_MockShStoreRecorder
}

// Recorder for MockShStore (not exported)
type _MockShStoreRecorder struct {
	mock *MockShStore
}

func NewMockShStore(ctrl *gomock.Controller) *MockShStore {
	mock := &MockShStore{ctrl: ctrl}
	mock.recorder = &_MockShStoreRecorder{mock}
	return mock
}

func (_m *MockShStore) EXPECT() *_MockShStoreRecorder {
	return _m.recorder
}

func (_m *MockShStore) CreateShTransaction(_param0 *sql.Tx, _param1 *ShTransaction) (*ShTransaction, error) {
	ret := _m.ctrl.Call(_m, "CreateShTransaction", _param0, _param1)
	ret0, _ := ret[0].(*ShTransaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateShTransaction(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateShTransaction", arg0, arg1)
}

func (_m *MockShStore) AddShTransactionItem(_param0 *sql.Tx, _param1 *ShTransaction, _param2 *ShTransactionItem) (*ShTransactionItem, error) {
	ret := _m.ctrl.Call(_m, "AddShTransactionItem", _param0, _param1, _param2)
	ret0, _ := ret[0].(*ShTransactionItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) AddShTransactionItem(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddShTransactionItem", arg0, arg1, arg2)
}

func (_m *MockShStore) GetShTransactionById(id int64, fetch_items bool) (*ShTransaction, error) {
	ret := _m.ctrl.Call(_m, "GetShTransactionById", id, fetch_items)
	ret0, _ := ret[0].(*ShTransaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetShTransactionById(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetShTransactionById", arg0, arg1)
}

func (_m *MockShStore) GetShTransactionSinceTransId(start_id int64) (int64, []*ShTransaction, error) {
	ret := _m.ctrl.Call(_m, "GetShTransactionSinceTransId", start_id)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].([]*ShTransaction)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (_mr *_MockShStoreRecorder) GetShTransactionSinceTransId(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetShTransactionSinceTransId", arg0)
}

func (_m *MockShStore) CreateItem(_param0 *ShItem) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "CreateItem", _param0)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateItem(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateItem", arg0)
}

func (_m *MockShStore) CreateItemInTx(_param0 *sql.Tx, _param1 *ShItem) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "CreateItemInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateItemInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateItemInTx", arg0, arg1)
}

func (_m *MockShStore) UpdateItemInTx(_param0 *sql.Tx, _param1 *ShItem) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "UpdateItemInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) UpdateItemInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "UpdateItemInTx", arg0, arg1)
}

func (_m *MockShStore) GetItemById(_param0 int64) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemById", _param0)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetItemById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemById", arg0)
}

func (_m *MockShStore) GetItemByIdInTx(_param0 *sql.Tx, _param1 int64) (*ShItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemByIdInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetItemByIdInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemByIdInTx", arg0, arg1)
}

func (_m *MockShStore) GetAllCompanyItems(_param0 int64) ([]*ShItem, error) {
	ret := _m.ctrl.Call(_m, "GetAllCompanyItems", _param0)
	ret0, _ := ret[0].([]*ShItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetAllCompanyItems(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetAllCompanyItems", arg0)
}

func (_m *MockShStore) CreateBranch(_param0 *ShBranch) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "CreateBranch", _param0)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateBranch(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateBranch", arg0)
}

func (_m *MockShStore) CreateBranchInTx(_param0 *sql.Tx, _param1 *ShBranch) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "CreateBranchInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateBranchInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateBranchInTx", arg0, arg1)
}

func (_m *MockShStore) UpdateBranchInTx(_param0 *sql.Tx, _param1 *ShBranch) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "UpdateBranchInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) UpdateBranchInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "UpdateBranchInTx", arg0, arg1)
}

func (_m *MockShStore) GetBranchById(_param0 int64) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "GetBranchById", _param0)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetBranchById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchById", arg0)
}

func (_m *MockShStore) GetBranchByIdInTx(_param0 *sql.Tx, _param1 int64) (*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "GetBranchByIdInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetBranchByIdInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchByIdInTx", arg0, arg1)
}

func (_m *MockShStore) ListCompanyBranches(_param0 int64) ([]*ShBranch, error) {
	ret := _m.ctrl.Call(_m, "ListCompanyBranches", _param0)
	ret0, _ := ret[0].([]*ShBranch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) ListCompanyBranches(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ListCompanyBranches", arg0)
}

func (_m *MockShStore) AddItemToBranch(_param0 *ShBranchItem) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "AddItemToBranch", _param0)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) AddItemToBranch(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddItemToBranch", arg0)
}

func (_m *MockShStore) AddItemToBranchInTx(_param0 *sql.Tx, _param1 *ShBranchItem) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "AddItemToBranchInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) AddItemToBranchInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddItemToBranchInTx", arg0, arg1)
}

func (_m *MockShStore) GetBranchItem(branch_id int64, item_id int64) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetBranchItem", branch_id, item_id)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetBranchItem(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchItem", arg0, arg1)
}

func (_m *MockShStore) GetBranchItemInTx(tnx *sql.Tx, branch_id int64, item_id int64) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetBranchItemInTx", tnx, branch_id, item_id)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetBranchItemInTx(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetBranchItemInTx", arg0, arg1, arg2)
}

func (_m *MockShStore) UpdateBranchItemInTx(_param0 *sql.Tx, _param1 *ShBranchItem) (*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "UpdateBranchItemInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) UpdateBranchItemInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "UpdateBranchItemInTx", arg0, arg1)
}

func (_m *MockShStore) GetItemsInBranch(_param0 int64) ([]*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemsInBranch", _param0)
	ret0, _ := ret[0].([]*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetItemsInBranch(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemsInBranch", arg0)
}

func (_m *MockShStore) GetItemsInAllCompanyBranches(_param0 int64) ([]*ShBranchItem, error) {
	ret := _m.ctrl.Call(_m, "GetItemsInAllCompanyBranches", _param0)
	ret0, _ := ret[0].([]*ShBranchItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetItemsInAllCompanyBranches(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetItemsInAllCompanyBranches", arg0)
}

func (_m *MockShStore) CreateCompany(u *User, c *Company) (*Company, error) {
	ret := _m.ctrl.Call(_m, "CreateCompany", u, c)
	ret0, _ := ret[0].(*Company)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateCompany(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateCompany", arg0, arg1)
}

func (_m *MockShStore) CreateCompanyInTx(_param0 *sql.Tx, _param1 *User, _param2 *Company) (*Company, error) {
	ret := _m.ctrl.Call(_m, "CreateCompanyInTx", _param0, _param1, _param2)
	ret0, _ := ret[0].(*Company)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateCompanyInTx(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateCompanyInTx", arg0, arg1, arg2)
}

func (_m *MockShStore) GetCompanyById(_param0 int64) (*Company, error) {
	ret := _m.ctrl.Call(_m, "GetCompanyById", _param0)
	ret0, _ := ret[0].(*Company)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetCompanyById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetCompanyById", arg0)
}

func (_m *MockShStore) CreateUser(u *User) (*User, error) {
	ret := _m.ctrl.Call(_m, "CreateUser", u)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateUser(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateUser", arg0)
}

func (_m *MockShStore) CreateUserInTx(tnx *sql.Tx, u *User) (*User, error) {
	ret := _m.ctrl.Call(_m, "CreateUserInTx", tnx, u)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) CreateUserInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateUserInTx", arg0, arg1)
}

func (_m *MockShStore) FindUserByName(_param0 string) (*User, error) {
	ret := _m.ctrl.Call(_m, "FindUserByName", _param0)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) FindUserByName(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindUserByName", arg0)
}

func (_m *MockShStore) FindUserById(_param0 int64) (*User, error) {
	ret := _m.ctrl.Call(_m, "FindUserById", _param0)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) FindUserById(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindUserById", arg0)
}

func (_m *MockShStore) FindUserByNameInTx(_param0 *sql.Tx, _param1 string) (*User, error) {
	ret := _m.ctrl.Call(_m, "FindUserByNameInTx", _param0, _param1)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) FindUserByNameInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindUserByNameInTx", arg0, arg1)
}

func (_m *MockShStore) SetUserPermission(_param0 *UserPermission) (*UserPermission, error) {
	ret := _m.ctrl.Call(_m, "SetUserPermission", _param0)
	ret0, _ := ret[0].(*UserPermission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) SetUserPermission(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetUserPermission", arg0)
}

func (_m *MockShStore) SetUserPermissionInTx(_param0 *sql.Tx, _param1 *UserPermission) (*UserPermission, error) {
	ret := _m.ctrl.Call(_m, "SetUserPermissionInTx", _param0, _param1)
	ret0, _ := ret[0].(*UserPermission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) SetUserPermissionInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetUserPermissionInTx", arg0, arg1)
}

func (_m *MockShStore) GetUserPermission(u *User, company_id int64) (*UserPermission, error) {
	ret := _m.ctrl.Call(_m, "GetUserPermission", u, company_id)
	ret0, _ := ret[0].(*UserPermission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) GetUserPermission(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetUserPermission", arg0, arg1)
}

func (_m *MockShStore) AddEntityRevisionInTx(_param0 *sql.Tx, _param1 *ShEntityRevision) (*ShEntityRevision, error) {
	ret := _m.ctrl.Call(_m, "AddEntityRevisionInTx", _param0, _param1)
	ret0, _ := ret[0].(*ShEntityRevision)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) AddEntityRevisionInTx(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddEntityRevisionInTx", arg0, arg1)
}

func (_m *MockShStore) GetRevisionsSince(start_from *ShEntityRevision) (int64, []*ShEntityRevision, error) {
	ret := _m.ctrl.Call(_m, "GetRevisionsSince", start_from)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].([]*ShEntityRevision)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (_mr *_MockShStoreRecorder) GetRevisionsSince(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetRevisionsSince", arg0)
}

func (_m *MockShStore) GetDataStore() DataStore {
	ret := _m.ctrl.Call(_m, "GetDataStore")
	ret0, _ := ret[0].(DataStore)
	return ret0
}

func (_mr *_MockShStoreRecorder) GetDataStore() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetDataStore")
}

func (_m *MockShStore) Begin() (*sql.Tx, error) {
	ret := _m.ctrl.Call(_m, "Begin")
	ret0, _ := ret[0].(*sql.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockShStoreRecorder) Begin() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Begin")
}
