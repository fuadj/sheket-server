package models

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
}

// implements ShDataStore
type shStore struct {
	DataStore
}

func NewShDataStore(ds DataStore) ShDataStore {
	store := &shStore{ds}
	return store
}

