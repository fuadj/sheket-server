package models

type User struct {
	UserId 		int64
	Username	string
	HashedPassword	string
}

type UserPermission struct {
	CompanyId 		int64
	UserId 			int64
	PermissionType	int64
	BranchId 		int64
}

func (b *shStore) CreateUser(*User) (*User, error) {
	return nil, nil
}

func (b *shStore) FindUserByName(username string) (*User, error) {
	return nil, nil
}

func (b *shStore) FindUserById(id int64) (*User, error) {
	return nil, nil
}

func (b *shStore) SetUserPermission(*UserPermission) (*UserPermission, error) {
	return nil, nil
}

func (b *shStore) GetUserPermission(*UserPermission) (*UserPermission, error) {
	return nil, nil
}