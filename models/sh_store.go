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
	GetShTransactionById(company_id, trans_id int64, fetch_items bool) (*ShTransaction, error)
	GetShTransactionByUUIDInTx(*sql.Tx, string) (*ShTransaction, error)

	GetShTransactionSinceTransId(company_id, start_id int64) (trans []*ShTransaction, err error)
}

type ItemStore interface {
	CreateItem(*ShItem) (*ShItem, error)
	CreateItemInTx(*sql.Tx, *ShItem) (*ShItem, error)

	UpdateItemInTx(*sql.Tx, *ShItem) (*ShItem, error)

	GetItemById(int64) (*ShItem, error)
	GetItemByUUIDInTx(*sql.Tx, string) (*ShItem, error)
	GetItemByIdInTx(*sql.Tx, int64) (*ShItem, error)
}

type BranchStore interface {
	CreateBranch(*ShBranch) (*ShBranch, error)
	CreateBranchInTx(*sql.Tx, *ShBranch) (*ShBranch, error)
	UpdateBranchInTx(*sql.Tx, *ShBranch) (*ShBranch, error)

	GetBranchByUUIDInTx(*sql.Tx, string) (*ShBranch, error)
	GetBranchById(int64) (*ShBranch, error)
	GetBranchByIdInTx(*sql.Tx, int64) (*ShBranch, error)
}

type BranchItemStore interface {
	AddItemToBranch(*ShBranchItem) (*ShBranchItem, error)
	AddItemToBranchInTx(*sql.Tx, *ShBranchItem) (*ShBranchItem, error)

	// the *ShBranchItem argument is only used to get the
	// branch and item id's
	GetBranchItem(branch_id, item_id int64) (*ShBranchItem, error)
	GetBranchItemInTx(tnx *sql.Tx, branch_id, item_id int64) (*ShBranchItem, error)
	UpdateBranchItemInTx(*sql.Tx, *ShBranchItem) (*ShBranchItem, error)
}

type CompanyStore interface {
	CreateCompany(u *User, c *Company) (*Company, error)

	// If the user doesn't exist, it will be created and then
	// the company gets created, it all happens in a single-transaction
	// NOTE: the transaction is not rolled-back in this method
	// The CALLER needs to rollback the transaction if error occurs
	CreateCompanyInTx(*sql.Tx, *User, *Company) (*Company, error)
	GetCompanyById(int64) (*Company, error)
}

type UserStore interface {
	CreateUser(u *User) (*User, error)
	CreateUserInTx(tnx *sql.Tx, u *User) (*User, error)

	FindUserByName(string) (*User, error)
	FindUserById(int64) (*User, error)

	FindUserByNameInTx(*sql.Tx, string) (*User, error)

	/**
	 * Permission is given to a user on a company basis.
	 * A permission typical looks like
	 * { company_id, user_id, permission }
	 */
	SetUserPermission(*UserPermission) (*UserPermission, error)

	// If a user is creating their own company, we need to make him/her
	// the admin of the company, that needs to happens in a single transaction with company creation
	// NOTE: the transaction is not rolled-back in this method
	// The CALLER needs to rollback the transaction if error occurs
	SetUserPermissionInTx(*sql.Tx, *UserPermission) (*UserPermission, error)

	GetUserPermission(u *User, company_id int64) (*UserPermission, error)

	GetUserCompanyPermissions(u *User) ([]*Pair_Company_UserPermission, error)

	GetCompanyMembersPermissions(c *Company) ([]*Pair_User_UserPermission, error)
}

type RevisionStore interface {
	AddEntityRevisionInTx(*sql.Tx, *ShEntityRevision) (*ShEntityRevision, error)

	// returns changes since the start revision
	GetRevisionsSince(start_from *ShEntityRevision) (latest_rev int64, since []*ShEntityRevision, err error)
}

type Source interface {
	GetDataStore() DataStore

	// used to start transactions
	// queries the DataStore
	Begin() (*sql.Tx, error)
}

type CategoryStore interface {
	CreateCategoryInTx(*sql.Tx, *ShCategory) (*ShCategory, error)
	GetCategoryById(int64) (*ShCategory, error)
	GetCategoryByIdInTx(*sql.Tx, int64) (*ShCategory, error)
	GetCategoryByUUIDInTx(*sql.Tx, string) (*ShCategory, error)

	UpdateCategoryInTx(*sql.Tx, *ShCategory) (*ShCategory, error)
}

type ShStore interface {
	TransactionStore
	ItemStore
	CategoryStore
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
