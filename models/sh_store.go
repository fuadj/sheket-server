package models

import "database/sql"

type TransactionStore interface {
	/**
	 * This assumes the transaction items are included in the transaction object.
	 * Creation of a {@code ShTransaction} happens only happens in a database
	 * transaction because either all or none of a user's business transactions
	 * should be committed.
	 */
	CreateShTransaction(*sql.Tx, *ShTransaction) (*ShTransaction, error)
	AddShTransactionItem(*sql.Tx, *ShTransaction, *ShTransactionItem) (*ShTransactionItem, error)

	// @args fetch_items 	whether you want the items in the transaction
	GetShTransactionById(id int64, fetch_items bool) (*ShTransaction, error)

	// this doesn't fetch items in the transaction
	// those need to be specifically queried
	GetShTransactionSinceTransId(int64) ([]*ShTransaction, error)
}

type ItemStore interface {
	CreateItem(*ShItem) (*ShItem, error)
	CreateItemInTx(*sql.Tx, *ShItem) (*ShItem, error)

	UpdateItemInTx(*sql.Tx, *ShItem) (*ShItem, error)

	GetItemById(int64) (*ShItem, error)
	GetItemByIdInTx(*sql.Tx, int64) (*ShItem, error)
	GetAllCompanyItems(int64) ([]*ShItem, error)
}

type BranchStore interface {
	CreateBranch(*ShBranch) (*ShBranch, error)
	CreateBranchInTx(*sql.Tx, *ShBranch) (*ShBranch, error)
	UpdateBranchInTx(*sql.Tx, *ShBranch) (*ShBranch, error)

	GetBranchById(int64) (*ShBranch, error)
	ListCompanyBranches(int64) ([]*ShBranch, error)
}

type BranchItemStore interface {
	AddItemToBranch(*ShBranchItem) (*ShBranchItem, error)
	AddItemToBranchInTx(*sql.Tx, *ShBranchItem) (*ShBranchItem, error)

	// the *ShBranchItem argument is only used to get the
	// branch and item id's
	GetBranchItem(branch_id, item_id int64) (*ShBranchItem, error)
	GetBranchItemInTx(tnx *sql.Tx, branch_id, item_id int64) (*ShBranchItem, error)
	UpdateBranchItemInTx(*sql.Tx, *ShBranchItem) (*ShBranchItem, error)

	GetItemsInBranch(int64) ([]*ShBranchItem, error)
	GetItemsInAllCompanyBranches(int64) ([]*ShBranchItem, error)
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

	GetUserPermission(u *User, company_id int64) (*UserPermission, error)
}

type RevisionStore interface {
	AddEntityRevisionInTx(*sql.Tx, *ShEntityRevision) (*ShEntityRevision, error)
	GetRevisionsSince(*ShEntityRevision) ([]*ShEntityRevision, error)
}

type ShStore interface {
	TransactionStore
	ItemStore
	BranchStore
	BranchItemStore
	CompanyStore
	UserStore
	RevisionStore

	GetDataStore() DataStore
}

// implements ShDataStore
type shStore struct {
	DataStore
}

func NewShDataStore(ds DataStore) ShStore {
	store := &shStore{ds}
	return store
}

func (s *shStore) GetDataStore() DataStore {
	return s.DataStore
}
