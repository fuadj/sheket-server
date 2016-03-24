package models

import (
	"database/sql"
	"fmt"
)

type BranchItemPair struct {
	BranchId int64
	ItemId   int64
}

// mocks BranchItemStore
type SimpleBranchItemStore struct {
	items      map[BranchItemPair]*ShBranchItem
	initialQty map[BranchItemPair]float64
}

func NewSimpleBranchItemStore(initialQuantity map[BranchItemPair]float64) *SimpleBranchItemStore {
	result := &SimpleBranchItemStore{}
	result.items = make(map[BranchItemPair]*ShBranchItem, 10)
	result.initialQty = make(map[BranchItemPair]float64, len(initialQuantity))
	for k, v := range initialQuantity {
		result.initialQty[k] = v
	}
	return result
}

func (m *SimpleBranchItemStore) AddItemToBranch(item *ShBranchItem) (*ShBranchItem, error) {
	return m.AddItemToBranchInTx(nil, item)
}

func (m *SimpleBranchItemStore) AddItemToBranchInTx(tnx *sql.Tx, item *ShBranchItem) (*ShBranchItem, error) {
	m.items[BranchItemPair{item.BranchId, item.ItemId}] = item
	return item, nil
}

func (m *SimpleBranchItemStore) GetBranchItem(branch_id, item_id int64) (*ShBranchItem, error) {
	return m.GetBranchItemInTx(nil, branch_id, item_id)
}

func (m *SimpleBranchItemStore) GetBranchItemInTx(tnx *sql.Tx, branch_id, item_id int64) (*ShBranchItem, error) {
	if item, ok := m.items[BranchItemPair{branch_id, item_id}]; ok {
		return item, nil
	} else if qty, ok := m.initialQty[BranchItemPair{branch_id, item_id}]; ok {
		return &ShBranchItem{
			BranchId: branch_id,
			ItemId:   item_id,
			Quantity: qty,
		}, nil
	}

	return nil, fmt.Errorf("item doesn't exist")
}

func (m *SimpleBranchItemStore) UpdateBranchItemInTx(tnx *sql.Tx, item *ShBranchItem) (*ShBranchItem, error) {
	m.items[BranchItemPair{item.BranchId, item.ItemId}] = item
	return item, nil
}

func (m *SimpleBranchItemStore) GetItemsInBranch(int64) ([]*ShBranchItem, error) {
	// TODO: not yet implemented
	return nil, nil
}
func (m *SimpleBranchItemStore) GetItemsInAllCompanyBranches(int64) ([]*ShBranchItem, error) {
	return nil, nil
}

type SimpleTransactionStore struct {
	Transactions map[int64]*ShTransaction
	TransItems   map[int64]map[*ShTransactionItem]bool
}

func NewSimpleTransactionStore() *SimpleTransactionStore {
	s := &SimpleTransactionStore{}
	s.Transactions = make(map[int64]*ShTransaction, 10)
	s.TransItems = make(map[int64]map[*ShTransactionItem]bool, 10)
	return s
}

func (s *SimpleTransactionStore) CreateShTransaction(tnx *sql.Tx, trans *ShTransaction) (*ShTransaction, error) {
	trans.TransactionId = int64(len(s.Transactions))
	s.Transactions[trans.TransactionId] = trans
	return trans, nil
}

func (s *SimpleTransactionStore) AddShTransactionItem(tnx *sql.Tx,
	trans *ShTransaction, trans_item *ShTransactionItem) (*ShTransactionItem, error) {
	trans_item.TransactionId = trans.TransactionId
	if !s.TransItems[trans_item.TransactionId][trans_item] {
		s.TransItems[trans_item.TransactionId][trans_item] = true
	}
	return trans_item, nil
}

func (s *SimpleTransactionStore) GetShTransactionById(id int64, fetch_items bool) (*ShTransaction, error) {
	trans, ok := s.Transactions[id]
	if !ok {
		return nil, fmt.Errorf("no transaction: %d", id)
	}
	if fetch_items {
		var items []*ShTransactionItem
		for item := range s.TransItems[id] {
			items = append(items, item)
		}
		trans.TransItems = items
	}
	return trans, nil
}

func (s *SimpleTransactionStore) GetShTransactionSinceTransId(int64) (int64, []*ShTransaction, error) {
	var trans []*ShTransaction
	var max_id int64
	for _, t := range s.Transactions {
		trans = append(trans, t)
		if t.TransactionId > max_id {
			max_id = t.TransactionId
		}
	}
	return max_id, trans, nil
}

type SimpleRevisionStore struct {
	Revisions []*ShEntityRevision
}

func NewSimpleRevisionStore(revs []*ShEntityRevision) *SimpleRevisionStore {
	s := &SimpleRevisionStore{}
	s.Revisions = revs
	return s
}

func (s *SimpleRevisionStore) AddEntityRevisionInTx(*sql.Tx, *ShEntityRevision) (*ShEntityRevision, error) {
	return nil, nil
}

func (s *SimpleRevisionStore) GetRevisionsSince(start_from *ShEntityRevision) (latest_rev int64, since []*ShEntityRevision, err error) {
	var max_rev int64
	if len(s.Revisions) > 0 {
		for _, rev := range s.Revisions {
			if rev.RevisionNumber > max_rev {
				max_rev = rev.RevisionNumber
			}
		}
	}
	return max_rev, s.Revisions, nil
}

