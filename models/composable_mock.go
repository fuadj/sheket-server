package models

import (
	"github.com/golang/mock/gomock"
)

// build this mock with parts that you mock out,
// helpful if you want to focus on a particular
// 'module', mock out the rest with a 'dummy' auto generate mock
type ComposableShStoreMock struct {
	TransactionStore
	ItemStore
	BranchStore
	BranchItemStore
	CompanyStore
	UserStore
	RevisionStore

	DataStore
}

func NewComposableShStoreMock(ctrl *gomock.Controller) *ComposableShStoreMock {
	c := &ComposableShStoreMock{}
	c.TransactionStore = NewMockTransactionStore(ctrl)
	c.ItemStore = NewMockItemStore(ctrl)
	c.BranchStore = NewMockBranchStore(ctrl)
	c.BranchItemStore = NewMockBranchItemStore(ctrl)
	c.CompanyStore = NewMockCompanyStore(ctrl)
	c.UserStore = NewMockUserStore(ctrl)
	c.RevisionStore = NewMockRevisionStore(ctrl)

	return c
}

func (c *ComposableShStoreMock) GetDataStore() DataStore {
	return c.DataStore
}
