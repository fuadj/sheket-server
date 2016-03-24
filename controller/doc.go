package controller

/*
	Transaction Sync Upload format
	{
		// NOTE: the user_id and company_id are not part of the body of
		// the upload but are part of the header. user_id is obviously stored
		// in a secure cookie. the company_id is sent raw.

		"transaction_rev":rev_number,
		"branch_item_rev":rev_number,

		"transactions": [
			// this is an array of transactions
			{
				// this is a negative value as an offline user is giving
				// a transaction a "temporary id" value until sync time
				// when it will be replaced with the global value
				// generated at the server.
				"trans_id":transaction_id

				// The only use of this id is to prevent possible duplicate
			// posting. This might happen if the user "upload's" their
				// changes and the server commits those changes, but the
				// connection with the user is cut before the server
				// could inform the user to update values with the sync'ed
				// values. So, when the user tries to "upload" again, we
				// need some way to tell that those were previously uploaded.
				"local_id":local_id

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

	Transaction Sync download format
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
		"update_local_transactions": [
			{
				"n":new_id, "o":old_id
			}, ...
		]
	}

	Entity Sync upload format
	{
		"item_rev":item_rev_number
		"branch_rev":branch_rev_number
		"branch_item_rev":branch_item_rev_number

		"types": [ changed entities {"items" | "branches" | "branch_items"} ]

		// could be {"items" OR "branches" OR "branch_items"}
		"entity": {
			// ids will be different for each type
			// e.g: it will be integer for "items" and "branches"
			//		but it will be a string of "branch_id:item_id" for branch item

			"create": [ ids ... ],
			"update": [ ids ... ],
			"delete": [ ids ... ],

			"fields": {
				// a map of "id" => objects affected
				// look above description to see what "id" means
				"id": {
					// look at -- INTERNAL FIELDS -- for details about fields of each entity
				},
				...
			}
		}
	}

	-- INTERNAL FIELDS -- of Entity Upload format
		"items":
			// these 2 fields are necessary for all CRUD operations
			// the rest is defined by each CRUD method
			"company_id: (int)
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

		"branches":
			// these 2 fields are necessary for all CRUD operations
			"company_id": (int)
			"branch_id": (int)
			// if it is create, "branch_id" should be negative for synchronization
			// see "items"."item_id" for more description

			// used on CREATE and UPDATE
			// if it is UPDATE, any missing fields are assumed to NOT change
			"name": (string)
			"location": (string)

		"branch_items":
			// these 3 fields are necessary for all CRUD operations
			"company_id": (int)
			// these 2 could be -ve, see descriptions of
			// "items"."item_id" & "branches"."branch_id"
			"item_id": (int)
			"branch_id": (int)

			// used on CREATE and UPDATE
			// if it is UPDATE, any missing fields are assumed to NOT change
			"quantity": (float)
			"item_location": (string)

*/
