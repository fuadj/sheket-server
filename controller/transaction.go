package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"io"
	"net/http"
	"sheket/server/controller/auth"
	"sheket/server/models"
	//"net/http/httputil"
)

const (
	no_rev = int64(-1) // this is a default 'nil' revision number

	// this is (upload|download) branch item revision number
	key_branch_item_rev  = "branch_item_rev"
	key_branch_item_sync = "sync_branch_items"

	// this is the downloaded(sent to user) newly created transactions
	key_updated_trans_ids = "updated_trans_ids"

	// this is the the key of the uploaded transaction array
	key_upload_transactions = "transactions"

	// this is (upload|download) transaction revision number
	key_trans_rev = "transaction_rev"

	// this is sent to the user if they have managerial privileges to
	// see transaction history
	key_sync_transactions = "sync_transactions"
)

type TransSyncData struct {
	UserTransRev      int64
	UserBranchItemRev int64
	NewTrans          []*models.ShTransaction
}

func parseTransactionPost(r io.Reader, info *IdentityInfo) (*TransSyncData, error) {
	data, err := simplejson.NewFromReader(r)
	if err != nil {
		return nil, err
	}

	trans_sync := &TransSyncData{}
	trans_sync.UserTransRev = data.Get(key_trans_rev).MustInt64(no_rev)
	trans_sync.UserBranchItemRev = data.Get(key_branch_item_rev).MustInt64(no_rev)

	if _, ok := data.CheckGet(key_upload_transactions); !ok {
		// there is nothing more to see!!!
		return trans_sync, nil
	}

	trans_arr := data.Get(key_upload_transactions).MustArray()
	trans_sync.NewTrans = make([]*models.ShTransaction, len(trans_arr))
	for i, v := range trans_arr {
		fields, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("malformed transactions array")
		}

		trans := &models.ShTransaction{}
		trans.TransactionId = toInt64(fields[models.TRANS_JSON_TRANS_ID])
		trans.BranchId = toInt64(fields[models.TRANS_JSON_BRANCH_ID])
		trans.Date = toInt64(fields[models.TRANS_JSON_DATE])
		if trans.ClientUUID, ok = fields[models.TRANS_JSON_UUID].(string); !ok {
			return nil, fmt.Errorf("transaction %d missing uuid", i)
		}

		trans.CompanyId = info.CompanyId
		trans.UserId = info.User.UserId

		items_arr, ok := fields[models.TRANS_JSON_ITEMS].([]interface{})
		if !ok {
			return nil, fmt.Errorf("items field non-existant in transaction")
		}
		trans_items := make([]*models.ShTransactionItem, len(items_arr))
		for j, v := range items_arr {
			fields, ok := v.([]interface{})
			if !ok {
				return nil, fmt.Errorf("malformed items array")
			} else if len(fields) < 4 {
				return nil, fmt.Errorf("not enought elements in items fields: '%v'", v)
			}

			var err error
			item := &models.ShTransactionItem{}
			item.CompanyId = info.CompanyId
			item.TransType, err = toIntErr(fields[0])
			if err != nil {
				return nil, err
			}
			item.ItemId, err = toIntErr(fields[1])
			if err != nil {
				return nil, err
			}
			item.OtherBranchId, err = toIntErr(fields[2])
			if err != nil {
				return nil, err
			}
			item.Quantity, err = toFloatErr(fields[3])
			if err != nil {
				return nil, err
			}
			trans_items[j] = item
		}
		trans.TransItems = trans_items

		trans_sync.NewTrans[i] = trans
	}

	return trans_sync, nil
}

/**
 * Since we will only be committing item changes only after every transaction
 * has been processed, we need to hold to the intermediate changes to the item
 * in memory. It is more efficient to update the object in memory than to
 * update it on datastore.
 */
type CachedBranchItem struct {
	*models.ShBranchItem

	itemExistsInBranch bool
	itemVisited        bool
}

/**
 * Searches the item if has already been used in a transaction before.
 * This is necessary because we are using the object in memory to track
 * changes made to the item. These changes will finally be committed
 * in a single update of the item in the datastore.
 */
func searchBranchItemInCache(tnx *sql.Tx, seenItems map[Pair_BranchItem]*CachedBranchItem,
	search_item *models.ShBranchItem) *CachedBranchItem {

	branch_id := search_item.BranchId
	item_id := search_item.ItemId

	if item, ok := seenItems[Pair_BranchItem{branch_id, item_id}]; ok {
		return item
	}

	// we've not found the item, so query datastore and add it to seenItems

	var cached_branch_item *CachedBranchItem
	branch_item, err := Store.GetBranchItemInTx(tnx,
		search_item.BranchId, search_item.ItemId)

	if err != nil { // the item doesn't exist in the branch-items list
		cached_branch_item = &CachedBranchItem{
			ShBranchItem:       search_item,
			itemExistsInBranch: false,
			itemVisited:        false}
	} else {
		cached_branch_item = &CachedBranchItem{
			ShBranchItem:       branch_item,
			itemExistsInBranch: true,
			itemVisited:        false}
	}

	cached_branch_item.itemVisited = false

	if !cached_branch_item.itemExistsInBranch {
		cached_branch_item.Quantity = float64(0)
	}

	seenItems[Pair_BranchItem{branch_id, item_id}] = cached_branch_item
	return cached_branch_item
}

type TransactionResult struct {
	OldId2New           map[int64]int64
	NewlyCreatedIds     map[int64]bool
	AffectedBranchItems map[Pair_BranchItem]*CachedBranchItem
}

func addTransactionsToDataStore(tnx *sql.Tx, new_transactions []*models.ShTransaction,
	company_id int64) (*TransactionResult, error) {

	result := &TransactionResult{}

	result.OldId2New = make(map[int64]int64, len(new_transactions))
	result.NewlyCreatedIds = make(map[int64]bool, len(new_transactions))

	result.AffectedBranchItems = make(map[Pair_BranchItem]*CachedBranchItem)
	for _, trans := range new_transactions {
		user_trans_id := trans.TransactionId

		/**
		 * If the transaction already exists, that must mean the user didn't
		 * get acknowledgement when posting and is trying to re-post, so just send
		 * them the id.
		 */
		prev_trans, t_err := Store.GetShTransactionByUUIDInTx(tnx, trans.ClientUUID)
		if t_err != nil {
			return nil, t_err
		} else if prev_trans != nil {
			result.OldId2New[user_trans_id] = prev_trans.TransactionId
			result.NewlyCreatedIds[prev_trans.TransactionId] = true
			continue
		}

		created, t_err := Store.CreateShTransactionInTx(tnx, trans)
		if t_err != nil {
			return nil, t_err
		}

		result.OldId2New[user_trans_id] = created.TransactionId
		result.NewlyCreatedIds[created.TransactionId] = true

		for _, trans_item := range trans.TransItems {
			branch_item := searchBranchItemInCache(tnx, result.AffectedBranchItems,
				&models.ShBranchItem{
					CompanyId: company_id, BranchId: trans.BranchId,
					ItemId: trans_item.ItemId,
				})
			branch_item.itemVisited = true

			switch trans_item.TransType {
			case models.TRANS_TYPE_ADD_PURCHASED,
				models.TRANS_TYPE_ADD_RETURN_ITEM:

				branch_item.Quantity += trans_item.Quantity

			case models.TRANS_TYPE_SUB_CURRENT_BRANCH_SALE:
				branch_item.Quantity -= trans_item.Quantity

			case models.TRANS_TYPE_SUB_DIRECT_SALE:
			// this doesn't affect inventory levels as the
			// items are being sold directly after purchase without
			// entering a store's inventory list. This is only
			// 'visible' on transaction history list to see who sold
			// how many

			// these 2 affect another branch
			// so, grab that branch and update it also
			case models.TRANS_TYPE_ADD_TRANSFER_FROM_OTHER,
				models.TRANS_TYPE_SUB_TRANSFER_TO_OTHER:

				other_branch_item := searchBranchItemInCache(tnx, result.AffectedBranchItems,
					&models.ShBranchItem{
						CompanyId: company_id, BranchId: trans_item.OtherBranchId,
						ItemId: trans_item.ItemId,
					})
				if !other_branch_item.itemVisited {
					other_branch_item.Quantity = float64(0)
				}
				other_branch_item.itemVisited = true

				if trans_item.TransType == models.TRANS_TYPE_ADD_TRANSFER_FROM_OTHER {
					branch_item.Quantity += trans_item.Quantity
					other_branch_item.Quantity -= trans_item.Quantity
				} else if trans_item.TransType == models.TRANS_TYPE_SUB_TRANSFER_TO_OTHER {
					branch_item.Quantity -= trans_item.Quantity
					other_branch_item.Quantity += trans_item.Quantity
				}
			}
		}
	}

	return result, nil
}

/**
 * Updates the items in branches. This works by updating the items in memory
 * until all transactions are processed, then finally committing the changes
 * into the datastore. This is more efficient than updating the item in the
 * datastore as soon as we see a transaction that affects it because a group
 * of transactions will affect a single item multiple times. So, do the
 * intermediate updates on the object in memory and only finally commit that
 * after all transactions are processed.
 *
 * the {@args changed_branch_items} is a map with key of Pair{branch_id, item_id}
 */
func updateBranchItems(tnx *sql.Tx, cached_branch_items map[Pair_BranchItem]*CachedBranchItem,
	company_id int64) error {

	for pair_branch_item, cached_item := range cached_branch_items {
		action_type := models.REV_ACTION_CREATE
		if cached_item.itemExistsInBranch {
			Store.UpdateBranchItemInTx(tnx, cached_item.ShBranchItem)
			action_type = models.REV_ACTION_UPDATE
		} else {
			Store.AddItemToBranchInTx(tnx, cached_item.ShBranchItem)
		}

		rev := &models.ShEntityRevision{
			CompanyId:        company_id,
			EntityType:       models.REV_ENTITY_BRANCH_ITEM,
			ActionType:       action_type,
			EntityAffectedId: pair_branch_item.BranchId,
			AdditionalInfo:   pair_branch_item.ItemId,
		}

		_, err := Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return err
		}
	}
	return nil
}

// used in testing
var currentUserGetter = auth.GetCurrentUser

func TransactionSyncHandler(w http.ResponseWriter, r *http.Request) {
	defer trace("TransactionSyncHandler")()
	/*
		d, err := httputil.DumpRequest(r, true)
		if err == nil {
			fmt.Printf("Request %s\n", string(d))
		}
	*/

	company_id := GetCurrentCompanyId(r)
	if company_id == INVALID_COMPANY_ID {
		writeErrorResponse(w, http.StatusNonAuthoritativeInfo)
		return
	}

	user, err := currentUserGetter(r)
	if err != nil {
		writeErrorResponse(w, http.StatusNonAuthoritativeInfo, err.Error())
		return
	}

	permission, err := Store.GetUserPermission(user, company_id)
	if err != nil { // the user doesn't have permission to post
		writeErrorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}

	info := &IdentityInfo{CompanyId: company_id, User: user, Permission: permission}
	posted_data, err := parseTransactionPost(r.Body, info)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	sync_result := make(map[string]interface{})
	sync_result[JSON_KEY_COMPANY_ID] = company_id

	var newly_created_trans_ids map[int64]bool

	// If the user just polled us to see if there were new
	// transactions without uploading new transactions,
	// the "key_new_transactions" will not exist in the response
	if len(posted_data.NewTrans) > 0 {
		tnx, err := Store.Begin()
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError)
			return
		}
		add_trans_result, err := addTransactionsToDataStore(tnx, posted_data.NewTrans, company_id)
		if err != nil {
			tnx.Rollback()
			writeErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		newly_created_trans_ids = add_trans_result.NewlyCreatedIds

		// update items affected by the transactions
		if err = updateBranchItems(tnx, add_trans_result.AffectedBranchItems, company_id); err != nil {
			tnx.Rollback()
			writeErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		tnx.Commit()

		i := int64(0)
		updated_ids := make([]map[string]int64, len(add_trans_result.OldId2New))
		for old_id, new_id := range add_trans_result.OldId2New {
			updated_ids[i] = map[string]int64{
				KEY_JSON_ID_OLD: old_id,
				KEY_JSON_ID_NEW: new_id,
			}
			i++
		}

		sync_result[key_updated_trans_ids] = updated_ids
	}

	// if user does have permission to see transaction history
	if permission.PermissionType <= models.PERMISSION_TYPE_BRANCH_MANAGER {
		max_trans_id, trans_history, err := fetchTransactionsSince(company_id,
			posted_data.UserTransRev, newly_created_trans_ids)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError)
			return
		}
		if len(trans_history) > 0 {
			sync_result[key_sync_transactions] = trans_history
		}
		sync_result[key_trans_rev] = max_trans_id
	}

	latest_rev, changed_branch_items, err := fetchChangedBranchItemsSinceRev(company_id,
		posted_data.UserBranchItemRev)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}

	sync_result[key_branch_item_rev] = latest_rev
	// if there are new changes to branch_items since last sync
	if len(changed_branch_items) > 0 {
		sync_result[key_branch_item_sync] = changed_branch_items
	}

	b, err := json.MarshalIndent(sync_result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	s := string(b)
	fmt.Printf("Transaction Sync response size:(%d)bytes\n%s\n\n\n", len(s), s)
}

func fetchTransactionsSince(company_id, trans_rev int64, newly_created_ids map[int64]bool) (store_max_rev int64,
	trans_since []map[string]interface{}, err error) {

	store_max_rev = trans_rev
	transactions, err := Store.GetShTransactionSinceTransId(company_id, trans_rev)
	if err != nil {
		return store_max_rev, nil, err
	}

	trans_history := make([]map[string]interface{}, len(transactions))
	i := 0
	for _, trans := range transactions {
		if trans.TransactionId > store_max_rev {
			store_max_rev = trans.TransactionId
		}
		// ignore currently added new transactions in the sync
		if newly_created_ids[trans.TransactionId] {
			continue
		}

		item_history := make([]map[string]interface{}, len(trans.TransItems))
		for j, trans_item := range trans.TransItems {
			item_history[j] = map[string]interface{}{
				"trans_type":   trans_item.TransType,
				"item_id":      trans_item.ItemId,
				"other_branch": trans_item.OtherBranchId,
				"quantity":     trans_item.Quantity,
			}
		}

		trans_history[i] = map[string]interface{}{
			models.TRANS_JSON_TRANS_ID:  trans.TransactionId,
			models.TRANS_JSON_UUID:      trans.ClientUUID,
			models.TRANS_JSON_USER_ID:   trans.UserId,
			models.TRANS_JSON_BRANCH_ID: trans.BranchId,
			models.TRANS_JSON_DATE:      trans.Date,
			models.TRANS_JSON_ITEMS:     item_history,
		}
		i++
	}
	return store_max_rev, trans_history[:i], nil
}

func fetchChangedBranchItemsSinceRev(company_id, branch_item_rev int64) (latest_revision int64,
	branch_items_since []map[string]interface{}, err error) {

	// this is guaranteed to return in ascending order till the latest
	max_rev, new_branch_item_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH_ITEM,
			RevisionNumber: branch_item_rev,
		})
	if err != nil {
		return max_rev, nil, err
	}

	result := make([]map[string]interface{}, len(new_branch_item_revs))
	i := 0
	for _, branch_rev := range new_branch_item_revs {
		branch_id := branch_rev.EntityAffectedId
		item_id := branch_rev.AdditionalInfo

		branch_item, err := Store.GetBranchItem(branch_id, item_id)
		if err != nil {
			continue
		}

		result[i] = map[string]interface{}{
			"branch_id": branch_id,
			"item_id":   item_id,
			"quantity":  branch_item.Quantity,
			"loc":       branch_item.ItemLocation,
		}
		i++
	}
	return max_rev, result[:i], nil
}
