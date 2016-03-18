package models

import (
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestCreateCompany(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_COMPANY)).
		WithArgs(company_name, company_contact).
		WillReturnRows(sqlmock.NewRows(_cols("company_id")).AddRow(company_id))
	mock.ExpectCommit()

	company := &Company{CompanyName: company_name, Contact: company_contact}
	company, err := store.CreateCompany(nil, company)
	if err != nil {
		t.Errorf("Company create error '%v'", err)
	}
}

func TestCreateCompanyFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_COMPANY)).
		WithArgs(company_name, company_contact).
		WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	company := &Company{CompanyName: company_name, Contact: company_contact}
	company, err := store.CreateCompany(nil, company)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestCreateCompanyInTransaction(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_COMPANY)).
		WithArgs(company_name, company_contact).
		WillReturnRows(sqlmock.NewRows(_cols("company_id")).AddRow(company_id))

	tnx, err := db.Begin()
	company := &Company{CompanyName: company_name, Contact: company_contact}
	company, err = store.CreateCompanyInTx(tnx, nil, company)
	if err != nil {
		t.Errorf("Company create error '%v'", err)
	}
}

func TestCreateCompanyInTransactionFail(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectBegin()
	mock.ExpectQuery(fmt.Sprintf("insert into %s", TABLE_COMPANY)).
		WithArgs(company_name, company_contact).
		WillReturnError(fmt.Errorf("insert error"))

	tnx, err := db.Begin()
	company := &Company{CompanyName: company_name, Contact: company_contact}
	company, err = store.CreateCompanyInTx(tnx, nil, company)
	if err == nil {
		t.Errorf("expected an error to occur")
	}
}

func TestFindCompanyById(t *testing.T) {
	mock_setup(t)
	defer mock_teardown()

	mock.ExpectQuery(fmt.Sprintf("select (.+) from %s", TABLE_COMPANY)).
		WithArgs(company_id).
		WillReturnRows(sqlmock.NewRows(_cols("company_id,company_name,contact")).
		AddRow(company_id, company_name, company_contact))

	company, err := store.GetCompanyById(company_id)
	if err != nil {
		t.Errorf("GetCompanyById error '%v'", err)
	} else if company == nil || company.CompanyId != company_id {
		t.Errorf("Invalid company")
	}
}
