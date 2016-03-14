package models

type ShItem struct {
	ItemId		int64
	CompanyId 		int64
	CategoryId		int64
	Name 			string
	ModelYear		string
	PartNumber		string
	BarCode 		string
	HasBarCode		bool
	ManualCode 		string
}

func (s *shStore) CreateItem(*ShItem) (*ShItem, error) {
	return nil, nil
}

func (s *shStore) GetItemById(int64) (*ShItem, error) {
	return nil, nil
}

func (s *shStore) GetAllCompanyItems(int64) ([]*ShItem, error) {
	return nil, nil
}
