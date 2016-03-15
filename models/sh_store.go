package models

import "database/sql"

var ShStore ShDataStore

type ShDataStore interface {
	CreateTransaction(*ShTransaction) (*ShTransaction, error)

	// whether you want the items in the transaction
	GetTransactionById(id int64, fetch_items_also bool) (*ShTransaction, error)

	// this doesn't fetch items in the transaction
	// those need to be specifically queried
	ListTransactionSinceTransId(int64) ([]*ShTransaction, error)

	CreateItem(*ShItem) (*ShItem, error)
	GetItemById(int64) (*ShItem, error)
	GetAllCompanyItems(int64) ([]*ShItem, error)

	CreateBranch(*ShBranch) (*ShBranch, error)
	GetBranchById(int64) (*ShBranch, error)
	ListCompanyBranches(int64) ([]*ShBranch, error)

	AddItemToBranch(*ShBranchItem) (*ShBranchItem, error)
	UpdateItemInBranch(*ShBranchItem) (*ShBranchItem, error)

	GetItemsInBranch(int64) ([]*ShBranchItem, error)
	GetItemsInAllCompanyBranches(int64) ([]*ShBranchItem, error)

	CreateCompany(u *User, c *Company) (*Company, error)

	// If the user doesn't exist, it will be created and then
	// the company gets created, it all happens in a single-transaction
	// NOTE: the transaction is not rolled-back in this method
	// The CALLER needs to rollback the transaction if error occurs
	CreateCompanyInTransaction(*sql.Tx, *User, *Company) (*Company, error)
	GetCompanyById(int64) (*Company, error)

	CreateUser(u *User, password string) (*User, error)
	CreateUserInTransaction(tnx *sql.Tx, u *User, password string) (*User, error)

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
	SetUserPermissionInTransaction(*sql.Tx, *UserPermission) (*UserPermission, error)

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