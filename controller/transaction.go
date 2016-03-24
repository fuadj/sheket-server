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
)

const (
	no_rev     = int64(-1) // this is a default 'nil' revision number
	invalid_id = int64(-1)

	s_key_user_id    = "user_id"
	s_key_company_id = "company_id"

	s_key_revision_transaction = "transaction_rev"
	s_key_revision_item        = "item_rev"
	s_key_revision_branch_item = "branch_item_rev"

	s_key_types = "types"

	s_key_created    = "create"
	s_key_updated    = "update"
	s_key_deleted    = "delete"
	s_key_attributes = "attributes"

	type_items        = "items"
	type_transactions = "transactions"
	type_branches     = "branches"
)

type TransSyncData struct {
	UserTransRev      int64
	UserBranchItemRev int64
	NewTrans          []*models.ShTransaction
}

func parseTransactionPost(r io.Reader) (*TransSyncData, error) {
	data, err := simplejson.NewFromReader(r)
	if err != nil {
		return nil, err
	}

	trans_sync := &TransSyncData{}
	trans_sync.UserTransRev = data.Get(s_key_revision_transaction).MustInt64(invalid_id)
	trans_sync.UserBranchItemRev = data.Get(s_key_revision_branch_item).MustInt64(invalid_id)

	if trans_json, ok := data.CheckGet(type_transactions); ok {
		trans_arr := trans_json.MustArray()
		trans_sync.NewTrans = make([]*models.ShTransaction, len(trans_arr))
		for i, v := range trans_arr {
			fields, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("malformed transactions array")
			}

			trans := &models.ShTransaction{}
			trans.TransactionId = _toInt64(fields["trans_id"])
			trans.LocalTransactionId = _toInt64(fields["local_id"])
			trans.BranchId = _toInt64(fields["branch_id"])
			trans.Date = _toInt64(fields["date"])

			items_arr, ok := fields["items"].([]interface{})
			if !ok {
				return nil, fmt.Errorf("items field non-existant in transaction")
			}
			trans_items := make([]*models.ShTransactionItem, len(items_arr))
			for j, v := range items_arr {
				fields, ok := v.([]interface{})
				if !ok {
					return nil, fmt.Errorf("malformed items array")
				}

				toInt := func(i interface{}) (int64, error) {
					var err error
					if val, ok := i.(json.Number); ok {
						var int_val int64
						int_val, err = val.Int64()
						if err == nil {
							return int_val, nil
						}
					}
					return 0, fmt.Errorf("'%v' not an integer", i)
				}

				toFloat := func(i interface{}) (float64, error) {
					var err error
					if val, ok := i.(json.Number); ok {
						var float_val float64
						float_val, err = val.Float64()
						if err == nil {
							return float_val, nil
						}
					}
					return 0, fmt.Errorf("'%v' not a float", i)
				}

				var err error
				item := &models.ShTransactionItem{}
				item.TransType, err = toInt(fields[0])
				if err != nil {
					return nil, err
				}
				item.ItemId, err = toInt(fields[1])
				if err != nil {
					return nil, err
				}
				item.OtherBranchId, err = toInt(fields[2])
				if err != nil {
					return nil, err
				}
				item.Quantity, err = toFloat(fields[3])
				if err != nil {
					return nil, err
				}
				trans_items[j] = item
			}
			trans.TransItems = trans_items

			trans_sync.NewTrans[i] = trans
		}
	}

	return trans_sync, nil
}

/**
 * Since we will only be committing item changes only after every transaction
 * has been processed, we need to hold to the intermediate changes to the item
 * in memory. It is more efficient to update the object in memory than to
 * update it on datastore.
 */
type SyncBranchItem struct {
	*models.ShBranchItem

	itemExistsInBranch bool
	// in combination with {@field itemExistsInBranch}, it is used
	// to "initialize" the branch item if the item doesn't exist in
	// the branch and we are seeing it for the first time
	firstTimeSeenItem bool
}

// Useful in map's as a key
// Without this, the key should be a 2-level thing
// e.g: map[outer_key]map[inner_key] object
type Pair_BranchItem struct {
	BranchId int64
	ItemId   int64
}

/**
 * Searches the item if has already been used in a transaction before.
 * This is necessary because we are using the object in memory to track
 * changes made to the item. These changes will finally be committed
 * in a single update of the item in the datastore.
 */
func searchBranchItemInMemory(tnx *sql.Tx, seenItems map[Pair_BranchItem]*SyncBranchItem,
	search_item *models.ShBranchItem) *SyncBranchItem {

	branch_id := search_item.BranchId
	item_id := search_item.ItemId

	if item, ok := seenItems[Pair_BranchItem{branch_id, item_id}]; ok {
		return item
	}

	// we've not found the item, so query datastore and add it to seenItems

	var sync_branch_item *SyncBranchItem
	branch_item, err := Store.GetBranchItemInTx(tnx,
		search_item.BranchId, search_item.ItemId)

	if err != nil { // the item doesn't exist in the branch-items list
		sync_branch_item = &SyncBranchItem{
			ShBranchItem:       search_item,
			itemExistsInBranch: false}
	} else {
		sync_branch_item = &SyncBranchItem{
			ShBranchItem:       branch_item,
			itemExistsInBranch: true}
	}

	sync_branch_item.firstTimeSeenItem = true
	if !sync_branch_item.itemExistsInBranch {
		sync_branch_item.Quantity = float64(0)
	}

	seenItems[Pair_BranchItem{branch_id, item_id}] = sync_branch_item
	return sync_branch_item
}

type TransactionResult struct {
	OldId2NewMap        map[int64]int64
	NewlyCreatedIds     map[int64]bool
	AffectedBranchItems map[Pair_BranchItem]*SyncBranchItem
}

func addTransactionsToDataStore(tnx *sql.Tx, new_transactions []*models.ShTransaction,
	company_id int64) (*TransactionResult, error) {

	result := &TransactionResult{}

	result.OldId2NewMap = make(map[int64]int64, len(new_transactions))
	result.NewlyCreatedIds = make(map[int64]bool, len(new_transactions))

	result.AffectedBranchItems = make(map[Pair_BranchItem]*SyncBranchItem)
	for _, trans := range new_transactions {
		user_trans_id := trans.TransactionId
		created, t_err := Store.CreateShTransaction(tnx, trans)
		if t_err != nil {
			return nil, fmt.Errorf(http.StatusText(http.StatusInternalServerError))
		}

		result.OldId2NewMap[user_trans_id] = created.TransactionId
		result.NewlyCreatedIds[created.TransactionId] = true

		for _, trans_item := range trans.TransItems {
			branch_item := searchBranchItemInMemory(tnx, result.AffectedBranchItems,
				&models.ShBranchItem{
					CompanyId: company_id, BranchId: trans.BranchId,
					ItemId: trans_item.ItemId,
				})

			switch trans_item.TransType {
			case models.TRANS_TYPE_ADD_PURCHASED_ITEM:
				branch_item.Quantity += trans_item.Quantity
			case models.TRANS_TYPE_SELL_CURRENT_BRANCH_ITEM:
				branch_item.Quantity -= trans_item.Quantity

			case models.TRANS_TYPE_SELL_PURCHASED_ITEM_DIRECTLY:
			// this doesn't affect inventory levels as the
			// items are being sold directly after purchase without
			// entering a store's inventory list. This is only
			// 'visible' on transaction history list to see who sold
			// how many

			// these 2 affect another branch
			// so, grab that branch and update it also
			case models.TRANS_TYPE_TRANSFER_OTHER_BRANCH_ITEM,
				models.TRANS_TYPE_SELL_OTHER_BRANCH_ITEM:

				other_branch_item := searchBranchItemInMemory(tnx, result.AffectedBranchItems,
					&models.ShBranchItem{
						CompanyId: company_id, BranchId: trans_item.OtherBranchId,
						ItemId: trans_item.ItemId,
					})
				if !other_branch_item.itemExistsInBranch {
					other_branch_item.Quantity = float64(0)
				}

				if trans_item.TransType == models.TRANS_TYPE_TRANSFER_OTHER_BRANCH_ITEM {
					branch_item.Quantity += trans_item.Quantity
					other_branch_item.Quantity -= trans_item.Quantity
				} else if trans_item.TransType == models.TRANS_TYPE_SELL_OTHER_BRANCH_ITEM {
					other_branch_item.Quantity -= trans_item.Quantity
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
func updateBranchItems(tnx *sql.Tx, changed_branch_items map[Pair_BranchItem]*SyncBranchItem,
	company_id int64) error {

	for pair_branch_item, sync_item := range changed_branch_items {
		action_type := int64(models.REV_ACTION_CREATE)
		if sync_item.itemExistsInBranch {
			Store.UpdateBranchItemInTx(tnx, sync_item.ShBranchItem)
			action_type = models.REV_ACTION_UPDATE
		} else {
			Store.AddItemToBranchInTx(tnx, sync_item.ShBranchItem)
		}

		rev := &models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH_ITEM,
			ActionType:     action_type,
			AffectedId:     pair_branch_item.BranchId,
			AdditionalInfo: pair_branch_item.ItemId,
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

	posted_data, err := parseTransactionPost(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	tnx, err := Store.Begin()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}
	add_trans_result, err := addTransactionsToDataStore(tnx, posted_data.NewTrans, company_id)
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	// update items affected by the transactions
	if err = updateBranchItems(tnx, add_trans_result.AffectedBranchItems, company_id); err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	tnx.Commit()

	i := int64(0)
	updated_ids := make([]map[string]int64, len(add_trans_result.NewlyCreatedIds))
	for old_id, new_id := range add_trans_result.OldId2NewMap {
		updated_ids[i] = map[string]int64{
			"o": old_id, "n": new_id,
		}
		i++
	}

	sync_result := make(map[string]interface{})

	sync_result["update_local_transactions"] = updated_ids

	// if he/she can see the transaction history
	if permission.PermissionType == models.U_PERMISSION_MANAGER {
		trans_history, err := fetchTransactionsSince(posted_data.UserTransRev)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError)
			return
		}
		sync_result["sync_trans"] = trans_history
	}

	changed_items, err := fetchChangedBranchItemsSinceRev(company_id, posted_data.UserBranchItemRev)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}
	sync_result["sync_branch_items"] = changed_items

	// TODO: write the updated revision numbers

	b, err := json.MarshalIndent(sync_result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func fetchTransactionsSince(trans_rev int64) ([]map[string]interface{}, error) {
	transactions, err := Store.GetShTransactionSinceTransId(trans_rev)
	if err != nil {
		return nil, err
	}

	trans_history := make([]map[string]interface{}, len(transactions))
	for i, trans := range transactions {
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
			"trans_id":  trans.TransactionId,
			"user_id":   trans.UserId,
			"branch_id": trans.BranchId,
			"date":      trans.Date,
			"items":     item_history,
		}
	}
	return trans_history, nil
}

func fetchChangedBranchItemsSinceRev(company_id, branch_item_rev int64) ([]map[string]interface{}, error) {
	new_branch_item_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH_ITEM,
			RevisionNumber: branch_item_rev,
		})
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(new_branch_item_revs))
	for i, branch_rev := range new_branch_item_revs {
		branch_id := branch_rev.AffectedId
		item_id := branch_rev.AdditionalInfo

		branch_item, err := Store.GetBranchItem(branch_id, item_id)
		if err != nil {
			// TODO: check if remove branch item was deleted in a transaction
			continue
		}

		result[i] = map[string]interface{}{
			"branch_id": branch_id,
			"item_id":   item_id,
			"quantity":  branch_item.Quantity,
			"loc":       branch_item.ItemLocation,
		}
	}
	return result, nil
}
