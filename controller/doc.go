package controller

/*
	Transaction Sync Upload format
	{
		// NOTE: the user_id and company_id are not part of the body of
		// the upload but are part of the header. user_id is obviously stored
		// in a secure cookie. the company_id is sent raw.

		"transaction_rev":rev_number,
		"branch_item_rev":rev_number,

		// this only exists if the user has created transactions!!
		"transactions": [
			// this is an array of transactions
			{
				// this is a temporary value assigned at the user's end
				// and replaced by a unique global value
				"trans_id":transaction_id

				// the branch the transactions originated in
				"branch_id":branch_id

				// date in long format
				"date":date

				"items": [
					[int, int, int, float], ...

					// the 1st int is transaction_type
					// the 2nd int is the item_id
					// the 3rd int is the other_branch_id
					// the 4th float is the quantity of the transaction
				]
		},
	}

	Transaction Sync response format
	{
		"transaction_rev":latest transaction revision
		"branch_item_rev":latest branch_item revision

		// this holds updated branch items since the user's "branch_item_rev"
		// this will only exist if there is there has been changes, otherwise it won't
		// exist, so the user should first check if their rev number and the
		// latest are different, and only then ask for "sync_branch_items"
		"sync_branch_items": [],

		// if this user has privileges to see transaction history,
		// it holds transactions since the user's "transaction_rev"
		// this will only exist if there is there has been changes, otherwise it won't
		// exist, so the user should first check if their rev number and the
		// latest are different, and only then ask for "sync_trans"
		"sync_trans": [],

		// this exists if the user has uploaded transactions
		// and the user's locally generated transaction id's
		// have been replaced with a global id at the server side.
		// those global id's are then sent to user so they may
		// update their local ids. If there was no new transactions
		// uploaded, this will not exist, it is to minimize network usage.
		"updated_trans_ids": [
			{
				"n":new_id, "o":old_id
			}, ...
		]
	}

	// The company id and user_id are sent in the header, and not in the json
	// This simplifies error checking even before parsing the request body
	Entity Sync upload format
	{
		"item_rev":item_rev_number
		"branch_rev":branch_rev_number
		"branch_item_rev":branch_item_rev_number

		"types": [ see "ENTITY TYPES" ]

		// could be {"items" OR "branches" OR "branch_items"}
		"entity": {
			// ids will be different for each type
			// look at -- ENTITY ID TYPES -- for details

			"create": [ ids ... ],
			"update": [ ids ... ],
			"delete": [ ids ... ],

			"fields": [
				// an array of entities
				// look at -- INTERNAL FIELDS -- for details about fields of each entity
				{		}, ...
			]
		}
	}


	-- ENTITY TYPES --
	1) "items"
	2) "members"
	3) "branches"
	4) "branch_items"
	-- END ENTITY TYPES --

	-- ENTITY ID TYPES -- of Entity Upload Format
	"items":
		id is integer
	"members":
		id is integer(it is the user's id)
	"branches":
		id is integer
	"branch_items":
		id is a colon separated string of branch_id & item_id => "branch_id:item_id"


	-- INTERNAL FIELDS -- of Entity Upload Format
		"items":
			// This is necessary for all CRUD types
			"item_id": (int)
			// the item_id should be negative if it is a create operation
			// the negative value will then be replaced with a global value
			// generated at the server. This ensures synchronization.

			// used on CREATE and UPDATE
			// if it is UPDATE, any missing fields are assumed to NOT change
			"name": (string)
			"model_year": (string)
			"part_number": (string)
			"bar_code": (string)
			"has_bar_code": (bool)
			"manual_code": (string)

		// TODO: not implemented yet
		"members":
			"user_id": (int)
			"permission": (string)

		"branches":
			// This is necessary for all CRUD types
			"branch_id": (int)
			// if it is create, "branch_id" should be negative for synchronization
			// see "items"."item_id" for more description

			// used on CREATE and UPDATE
			// if it is UPDATE, any missing fields are assumed to NOT change
			"name": (string)
			"location": (string)

		"branch_items":
			// The "id" field can contain -ve values is either branch|item is being created.
			// see "items"."item_id" & "branches"."branch_id" for more info
			// The "id" and & "company_id" fields are necessary.
			"id": (string) 	// "branch_id:item_id"

			// used on CREATE and UPDATE
			// if it is UPDATE, any missing fields are assumed to NOT change
			"quantity": (float)
			"item_location": (string)

	Entity Sync Result format
	{
		"company_id":company_id
		"item_rev":item_rev
		"branch_rev": branch_rev

		// only exists if user uploaded new items and server assigned new ids to them
		"updated_item_ids": [
			{"o": old_id, "n":new_id}, ...
		]

		// only exists if user uploaded new branches and server assigned new ids to them
		"updated_branch_ids": [
			{"o": old_id, "n":new_id}, ...
		]

		// only exists if there are added|changed items
		"sync_items": [
			{
				"item_id":,
				"item_name":,
				"model_year":,
				"part_number":,
				"bar_code":,
				"manual_code":,
				"has_bar_code":
			}, ...
		]

		// TODO: not implemented yet
		// only exists if there are changed members
		"sync_members": [
			{
				"user_id":,
				"user_name":,
				"permission":
			}, ...
		]

		// only exists if there are added|changed branches
		"sync_branches": [
			{
				"branch_id":,
				"name":,
				"location":,
			}, ...
		]
	}

	User Sign-up Format
	{
		"username":username
		"password":password
	}

	User Sign-up Response
	{
		"username":username
		"new_user_id":user_id
	}
*/
