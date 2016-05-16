package models

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

// table names
const (
	TABLE_USER             = "s_user_table"
	TABLE_COMPANY          = "s_company"
	TABLE_BRANCH           = "s_branch"
	TABLE_U_PERMISSION     = "s_user_permission_table"
	TABLE_CATEGORY         = "s_category"
	TABLE_INVENTORY_ITEM   = "s_inventory_item"
	TABLE_BRANCH_ITEM      = "s_branch_item"
	TABLE_TRANSACTION      = "s_business_transaction"
	TABLE_TRANSACTION_ITEM = "s_business_transaction_item"
	TABLE_ENTITY_REVISION  = "s_table_entity_revision"
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
	db, err := sql.Open("postgres", "user=postgres password=abcdabcd dbname=sheket sslmode=disable")
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
		// fall-through when error occurs, to catch it at the each
		if err != nil {
			return
		}

		_, err = db.Exec(q, args...)
		if err != nil {
			fmt.Printf("'%s' '%s'", q, err.Error())
			return
		}
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
		"UNIQUE(company_name));", TABLE_COMPANY))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// user-permission-table
		"company_id			INTEGER REFERENCES %s(company_id), "+
		"user_id			INTEGER REFERENCES %s(user_id), "+
		"permission			VARCHAR(1000) NOT NULL);",
		TABLE_U_PERMISSION, TABLE_COMPANY, TABLE_USER))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// branch-table
		"branch_id		SERIAL PRIMARY KEY, "+
		"client_uuid	uuid, "+
		"company_id		INTEGER REFERENCES %s(company_id), "+
		"branch_name	VARCHAR(260) NOT NULL, "+
		"location 		VARCHAR(200), "+

		"UNIQUE(company_id, branch_name));",
		TABLE_BRANCH, TABLE_COMPANY))

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// category table
		"category_id	SERIAL PRIMARY KEY, "+
		"client_uuid	uuid, "+
		"company_id		INTEGER REFERENCES %s(company_id), "+
		"name			VARCHAR(100) NOT NULL, "+
		"parent_id		INTEGER REFERENCES %s(category_id));",
		TABLE_CATEGORY, TABLE_COMPANY, TABLE_CATEGORY))
	if err = checkRootCategoryCreated(db); err != nil {
		return nil, err
	}

	exec(t_name("CREATE TABLE IF NOT EXISTS %s ( "+
		// item table
		"item_id		SERIAL PRIMARY KEY, "+
		"client_uuid 	uuid, "+
		"company_id		INTEGER REFERENCES %s(company_id), "+
		"category_id	INTEGER REFERENCES %s(category_id) ON DELETE SET DEFAULT, "+
		"name			VARCHAR(200) NOT NULL, "+
		"model_year		VARCHAR(10), "+
		"part_number	VARCHAR(30), "+
		"bar_code		VARCHAR(30), "+
		"has_bar_code	BOOL, "+
		"manual_code	VARCHAR(30));",
		TABLE_INVENTORY_ITEM, TABLE_COMPANY, TABLE_CATEGORY))
	// set the "root category" as default to be used "on delete clause"
	if _, err := db.Exec(
		fmt.Sprintf("ALTER TABLE %s ALTER COLUMN category_id set default %d",
			TABLE_INVENTORY_ITEM, ROOT_CATEGORY_ID)); err != nil {
		return nil, err
	}

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
	 */
	exec(t_name("create table if not exists %s ( "+
		// transaction-table
		"transaction_id			SERIAL PRIMARY KEY, "+
		"client_uuid			uuid, "+
		"company_id				integer references %s(company_id), "+
		"branch_id				integer references %s(branch_id), "+
		"user_id				integer references %s(user_id), "+
		"t_date 					integer);",
		TABLE_TRANSACTION, TABLE_COMPANY, TABLE_BRANCH, TABLE_USER))

	/**
	 * Transaction items looks like
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
		// transaction-items table
		"company_id			integer references %s(company_id), "+
		"transaction_id 	INTEGER REFERENCES %s(transaction_id), "+
		"trans_type			INTEGER NOT NULL, "+
		"item_id			INTEGER REFERENCES %s(item_id), "+
		"other_branch_id 	INTEGER, "+
		"quantity 			REAL NOT NULL);",
		TABLE_TRANSACTION_ITEM, TABLE_COMPANY, TABLE_TRANSACTION, TABLE_INVENTORY_ITEM))

	exec(t_name("create table if not exists %s ( "+
		"company_id			integer references %s(company_id), "+
		"revision_number 	integer not null, "+
		"entity_type 		integer not null, "+
		"action_type 		integer not null, "+
		"affected_id 		integer not null, "+
		"additional_info 	integer);",
		TABLE_ENTITY_REVISION, TABLE_COMPANY))

	if err != nil {
		return nil, err
	}

	return &dbStore{db}, nil
}

func checkRootCategoryCreated(db *sql.DB) error {
	rows, err := db.Query(
		fmt.Sprintf("select category_id from %s where category_id = $1", TABLE_CATEGORY),
		ROOT_CATEGORY_ID)
	if err != nil {
		return err
	}
	if !rows.Next() {
		rows.Close()
		if _, err = db.Exec(
			fmt.Sprintf("insert into %s (name) values ($1);", TABLE_CATEGORY),
			ROOT_CATEGORY_NAME); err != nil {
			return err
		}
		rows, err = db.Query(
			fmt.Sprintf("select category_id from %s where category_id = $1", TABLE_CATEGORY),
			ROOT_CATEGORY_ID)
		if err != nil {
			return err
		}
		if !rows.Next() {
			return fmt.Errorf("root category creation failed")
		}
		rows.Close()
	} else {
		rows.Close()
	}

	return nil
}
