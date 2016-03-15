package models

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

// table names
const (
	TABLE_USER              = "user_table"
	TABLE_COMPANY           = "company"
	TABLE_BRANCH            = "branch"
	TABLE_CATEGORY          = "category"
	TABLE_U_PERMISSION      = "user_permission_table"
	TABLE_INVENTORY_ITEM    = "inventory_item"
	TABLE_BRANCH_ITEM       = "branch_item"
	TABLE_TRANSACTION       = "business_transaction"
	TABLE_TRANSACTION_ELEM  = "business_transaction_elem"
)

var (
	GlobalDS DataStore
)


// Objects that implement this interface can be used as
// a store for data in Sheket.
type DataStore interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)

	// Begin a transaction
	Begin() (*sql.Tx, error)
}

// Implements DataStore under a database implementation
type dbStore struct {
	db *sql.DB
}

func (d *dbStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.db.Exec(query, args...)
}

func (d *dbStore) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}

func (d *dbStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

func (d *dbStore) Begin() (*sql.Tx, error) {
	return d.db.Begin()
}

func ConnectDbStore() (*dbStore, error) {
	db, err := sql.Open("postgres", "user=postgres password=abcdabcd dbname=fastsale sslmode=disable")
	if err != nil {
		return nil, err
	}

	// cleanup function
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	exec := func(q string, args ...interface{}) {
		if err != nil {
			return
		}
		_, err = db.Exec(q, args...)
	}

	// TODO: make this more robust
	t_name := fmt.Sprintf

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// user-table
		"user_id		SERIAL PRIMARY KEY, "+
		"username		VARCHAR(100) NOT NULL, "+
		"hashpass 		VARCHAR(260) NOT NULL, "+
		"UNIQUE(username));", TABLE_USER))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// company-table
		"company_id		SERIAL PRIMARY KEY, "+
		"company_name	VARCHAR(100) NOT NULL, "+
		"contact		VARCHAR(260) NOT NULL, "+
		"UNIQUE(name));", TABLE_COMPANY))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// branch-table
		"branch_id		SERIAL PRIMARY KEY, "+
		"company_id		INTEGER REFERENCES %s(company_id), "+
		"branch_name	VARCHAR(260) NOT NULL, "+
		"location 		VARCHAR(200), " +

		"UNIQUE(company_id, branch_name));",
		TABLE_BRANCH, TABLE_COMPANY))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// category_table
		"company_id		INTEGER REFERENCES %s(company_id), "+
		"category_id	INTEGER NOT NULL, "+
		"name			VARCHAR(200));",
		TABLE_CATEGORY, TABLE_COMPANY))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// user-permission-table
		"company_id			INTEGER REFERENCES %s(company_id), "+
		"user_id			INTEGER REFERENCES %s(user_id), "+
		"permission_type	INTEGER NOT NULL, "+

		// This is optional, the user could be restricted
		// to a particular branch or not.
		// It all depends on the permission_type
		"branch_id			INTEGER);",
		TABLE_U_PERMISSION, TABLE_COMPANY, TABLE_USER))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// item table
		"item_id		INTEGER NOT NULL, "+
		"company_id		INTEGER REFERENCES %s(id), "+
		"category_id	INTEGER NOT NULL, "+
		"name			VARCHAR(200) NOT NULL, "+
		"model_year		VARCHAR(10), "+
		"part_number	VARCHAR(30), "+
		"bar_code		VARCHAR(30), "+
		"has_bar_code	BOOL, "+
		"manual_code	VARCHAR(30));",
		TABLE_INVENTORY_ITEM, TABLE_COMPANY))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// branch-item table
		"company_id		INTEGER REFERENCES %s(company_id), "+
		"branch_id		INTEGER NOT NULL, "+
		"item_id		INTEGER NOT NULL, "+
		"quantity		REAL NOT NULL, "+
		"item_location		VARCHAR(20), "+
		"unique(branch_id, item_id));",
		TABLE_BRANCH_ITEM, TABLE_COMPANY))

	/**
	 * A Transaction looks like
	 * { company_id, transaction_id, user_id, date }
	 * {@column transaction_id} is a unique number across the transactions of a company
	 * {@column user_id} is the person who performed the transaction, globally unique
	 */
	exec(t_name("create table if not exists %s ( "+
		// transaction-table
		"transaction_id	integer not null, "+
		"company_id		integer references %s(company_id), "+
		"branch_id		integer references %s(branch_id), "+
		"user_id		integer references %s(user_id), "+
		"date 			integer, "+
		"unique(company_id, transaction));",
		TABLE_TRANSACTION, TABLE_COMPANY, TABLE_BRANCH, TABLE_USER))

	/**
	 * Transaction elems looks like
	 * { transaction_id, trans_type, item_id, other_branch_id, quantity }
	 * {@column transaction_id} is a foreign key into transaction table
	 * {@column trans_type} tells what type of transaction it is
	 * 		it couldn't be placed in the transactions table because a single
	 *		transaction might involve many types.
	 * 		e.g:
	 *			if a store sells 10 laptops and 3 printers
	 *			and if the laptops were in the shop's store
	 *			but the printers were not available in the store
	 *			and so the shop bought the printers from a neighbour store
	 *			and sold them to its customer.
	 *
	 *			so, the laptops affect the inventory of the shop BUT the printers don't.
	 *			{@column trans_type} is a defined set of these possible transactions types.
	 * {@column item_id} is the item that was sold
	 * {@column other_branch_id} if the transaction affects the inventory of
	 * 						other branches, (e.g: if it mentions warehouse inventory this will be the warehouse id}
	 * {@column quantity} is the number of {@column item_id} in the transaction
	 */
	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// transaction-elems table
		"transaction_id 	INTEGER REFERENCES %s(transaction_id), "+
		"trans_type			INTEGER NOT NULL, "+
		"item_id			INTEGER REFERENCES %s(item_id), "+
		"other_branch_id 	INTEGER, "+
		"quantity 			REAL NOT NULL));",
		TABLE_TRANSACTION_ELEM, TABLE_TRANSACTION, TABLE_INVENTORY_ITEM))

	if err != nil {
		return nil, err
	}

	return &dbStore{db}, nil
}
