package models

import (
	"database/sql"
	"errors"
)

// ErrNoData is the error returned if there is not data available
// for your "query". You should check the returned error to see
// if this is the type. If it ain't that means shit has gone wrong.
var ErrNoData = errors.New("sheket: no data found")

type TransactionStore interface {
	CreateShTransactionInTx(*sql.Tx, *ShTransaction) (*ShTransaction, error)
	AddShTransactionItemInTx(*sql.Tx, *ShTransaction, *ShTransactionItem) (*ShTransactionItem, error)

	// @args fetch_items 	whether you want the items in the transaction
	GetShTransactionById(company_id int, trans_id int64, fetch_items bool) (*ShTransaction, error)
	GetShTransactionByUUIDInTx(*sql.Tx, string) (*ShTransaction, error)

	GetShTransactionSinceTransId(company_id int, prev_trans_id int64) (trans []*ShTransaction, err error)
}

type ItemStore interface {
	CreateItemInTx(*sql.Tx, *ShItem) (*ShItem, error)

	UpdateItemInTx(*sql.Tx, *ShItem) (*ShItem, error)

	GetItemById(int) (*ShItem, error)
	GetItemByUUIDInTx(*sql.Tx, string) (*ShItem, error)
	GetItemByIdInTx(*sql.Tx, int) (*ShItem, error)
}

type BranchStore interface {
	CreateBranchInTx(*sql.Tx, *ShBranch) (*ShBranch, error)
	UpdateBranchInTx(*sql.Tx, *ShBranch) (*ShBranch, error)

	GetBranchByUUIDInTx(*sql.Tx, string) (*ShBranch, error)
	GetBranchById(int) (*ShBranch, error)
	GetBranchByIdInTx(*sql.Tx, int) (*ShBranch, error)
}

type BranchItemStore interface {
	AddItemToBranchInTx(*sql.Tx, *ShBranchItem) (*ShBranchItem, error)

	// the *ShBranchItem argument is only used to get the
	// branch and item id's
	GetBranchItem(branch_id, item_id int) (*ShBranchItem, error)
	GetBranchItemInTx(tnx *sql.Tx, branch_id, item_id int) (*ShBranchItem, error)
	UpdateBranchItemInTx(*sql.Tx, *ShBranchItem) (*ShBranchItem, error)
}

type CompanyStore interface {
	// If the user doesn't exist, it will be created and then
	// the company gets created, it all happens in a single-transaction
	// NOTE: the transaction is not rolled-back in this method
	// The CALLER needs to rollback the transaction if error occurs
	CreateCompanyInTx(*sql.Tx, *User, *Company) (*Company, error)
	GetCompanyById(int) (*Company, error)

	UpdateCompanyInTx(*sql.Tx, *Company) (*Company, error)
}

type UserStore interface {
	CreateUserInTx(tnx *sql.Tx, u *User) (*User, error)

	FindUserById(int) (*User, error)

	// searches for the user by the unique id given to the user by the provider
	FindUserWithProviderIdInTx(tnx *sql.Tx, provider_id int, provider_user_id string) (*User, error)

	UpdateUserInTx(*sql.Tx, *User) (*User, error)

	RemoveUserFromCompanyInTx(tnx *sql.Tx, user_id, company_id int) (error)

	// If a user is creating their own company, we need to make him/her
	// the admin of the company, that needs to happens in a single transaction with company creation
	// NOTE: the transaction is not rolled-back in this method
	// The CALLER needs to rollback the transaction if error occurs
	SetUserPermissionInTx(*sql.Tx, *UserPermission) (*UserPermission, error)

	GetUserPermission(u *User, company_id int) (*UserPermission, error)

	GetUserCompanyPermissions(u *User) ([]*Pair_Company_UserPermission, error)

	GetCompanyMembersPermissions(c *Company) ([]*Pair_User_UserPermission, error)
}

type RevisionStore interface {
	AddEntityRevisionInTx(*sql.Tx, *ShEntityRevision) (*ShEntityRevision, error)

	// returns changes since the start revision
	GetRevisionsSince(start_from *ShEntityRevision) (latest_rev int, since []*ShEntityRevision, err error)
}

type Source interface {
	GetDataStore() DataStore

	// used to start transactions
	// queries the DataStore
	Begin() (*sql.Tx, error)
}

type CategoryStore interface {
	CreateCategoryInTx(*sql.Tx, *ShCategory) (*ShCategory, error)
	GetCategoryById(int) (*ShCategory, error)
	GetCategoryByIdInTx(*sql.Tx, int) (*ShCategory, error)
	GetCategoryByUUIDInTx(*sql.Tx, string) (*ShCategory, error)

	UpdateCategoryInTx(*sql.Tx, *ShCategory) (*ShCategory, error)
	DeleteCategoryInTx(*sql.Tx, int) (error)
}

type BranchCategoryStore interface {
	AddCategoryToBranchInTx(*sql.Tx, *ShBranchCategory) (*ShBranchCategory, error)

	GetBranchCategory(branch_id, category_id int) (*ShBranchCategory, error)
	GetBranchCategoryInTx(tnx *sql.Tx, branch_id, category_id int) (*ShBranchCategory, error)

	DeleteBranchCategoryInTx(tnx *sql.Tx, branch_id, category_id int) (error)
}

type ShStore interface {
	TransactionStore
	ItemStore
	CategoryStore
	BranchCategoryStore
	BranchStore
	BranchItemStore
	CompanyStore
	UserStore
	RevisionStore

	Source
}

// implements ShStore
type shStore struct {
	DataStore
}

func NewShStore(ds DataStore) ShStore {
	store := &shStore{ds}
	return store
}

func (s *shStore) GetDataStore() DataStore {
	return s.DataStore
}

func (s *shStore) Begin() (*sql.Tx, error) {
	return s.DataStore.Begin()
}
