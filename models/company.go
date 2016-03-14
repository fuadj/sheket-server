package models

type Company struct {
	CompanyId 		int64
	CompanyName		string
	Contact 		string
}

func (b *shStore) CreateCompany(*User, *Company) (*Company, error) {
	return nil, nil
}

func (b *shStore) FindCompanyById(id int64) (*Company, error) {
	return nil, nil
}