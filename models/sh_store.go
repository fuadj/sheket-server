package models

import "database/sql"

type ShDataStore interface {
	GetDataStore() DataStore

	/**
	 * This assumes the transaction items are included in the transaction object.
	 * Creation of a {@code ShTransaction} happens only happens in a database
	 * transaction because either all or none of a user's business transactions
	 * should be committed.
	 */
	CreateShTransaction(*sql.Tx, *ShTransaction) (*ShTransaction, error)
	AddShTransactionElem(*sql.Tx, *ShTransaction, *ShTransactionItem) (*ShTransactionItem, error)

	// @args fetch_items 	whether you want the items in the transaction
	GetShTransactionById(id int64, fetch_items bool) (*ShTransaction, error)

	// this doesn't fetch items in the transaction
	// those need to be specifically queried
	ListShTransactionSinceTransId(int64) ([]*ShTransaction, error)

	CreateItem(*ShItem) (*ShItem, error)
	CreateItemInTx(*sql.Tx, *ShItem) (*ShItem, error)

	GetItemById(int64) (*ShItem, error)
	GetAllCompanyItems(int64) ([]*ShItem, error)

	CreateBranch(*ShBranch) (*ShBranch, error)
	CreateBranchInTx(*sql.Tx, *ShBranch) (*ShBranch, error)

	GetBranchById(int64) (*ShBranch, error)
	ListCompanyBranches(int64) ([]*ShBranch, error)

	AddItemToBranch(*ShBranchItem) (*ShBranchItem, error)
	UpdateItemInBranch(*sql.Tx, *ShBranchItem) (*ShBranchItem, error)

	GetItemsInBranch(int64) ([]*ShBranchItem, error)
	GetItemsInAllCompanyBranches(int64) ([]*ShBranchItem, error)

	CreateCompany(u *User, c *Company) (*Company, error)

	// If the user doesn't exist, it will be created and then
	// the company gets created, it all happens in a single-transaction
	// NOTE: the transaction is not rolled-back in this method
	// The CALLER needs to rollback the transaction if error occurs
	CreateCompanyInTx(*sql.Tx, *User, *Company) (*Company, error)
	GetCompanyById(int64) (*Company, error)

	CreateUser(u *User, password string) (*User, error)
	CreateUserInTx(tnx *sql.Tx, u *User, password string) (*User, error)

	FindUserByName(string) (*User, error)
	FindUserById(int64) (*User, error)

	/**
	 * Permission is given to a user on a company basis.
	 * A permission typical looks like
	 * { company_id, user_id, permission_type, [optional branch_id] }
	 * if the permission_type requires a user be given permission for a
	 * specific branch only, then the branch_id will be used.
	 */
	SetUserPermission(*UserPermission) (*UserPermission, error)

	// If a user is creating their own company, we need to make him/her
	// the admin of the company, that needs to happens in a single transaction with company creation
	// NOTE: the transaction is not rolled-back in this method
	// The CALLER needs to rollback the transaction if error occurs
	SetUserPermissionInTx(*sql.Tx, *UserPermission) (*UserPermission, error)

	// Pass in the user_id and company_id and you will get
	// the full information about the user permission in that
	// company if it exists.
	GetUserPermission(*UserPermission) (*UserPermission, error)
}

// implements ShDataStore
type shStore struct {
	DataStore
}

func NewShDataStore(ds DataStore) ShDataStore {
	store := &shStore{ds}
	return store
}

func (s *shStore) GetDataStore() DataStore {
	return s.DataStore
}
