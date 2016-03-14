package models

type ShTransaction struct {
	CompanyId		int64
	TransactionId	int64
	UserId 			int64
	Date			int64
	TransElems		*[]ShTransactionElem
}

type ShTransactionElem struct {
	TransactionId	int64
	TransType		int64
	ItemId			int64
	BranchId		int64
	OtherBranchId	int64
	Quantity		float64
}

func (s *ShTransactionElem) Map() map[string]interface{} {
	result := make(map[string]interface{})
	result["transaction_id"] = s.TransactionId
	result["trans_type"] = s.TransType
	result["item_id"] = s.ItemId
	result["branch_id"] = s.BranchId
	result["other_branch"] = s.OtherBranchId
	result["quantity"] = s.Quantity
	return result
}

func (s *shStore) CreateTransaction(*ShTransaction) (*ShTransaction, error) {
	return nil, nil
}

func (s *shStore) GetTransactionById(int64, bool) (*ShTransaction, error) {
	return nil, nil
}

func (s *shStore) ListTransactionSinceTransId(int64) ([]*ShTransaction, error) {
	return nil, nil
}
