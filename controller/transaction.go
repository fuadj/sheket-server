package controller

import (
	"database/sql"
	"golang.org/x/net/context"
	"sheket/server/models"
	sp "sheket/server/sheketproto"
)

type Pair_BranchItem struct {
	BranchId int
	ItemId   int
}

type AffectedBranchItems map[Pair_BranchItem]*CachedBranchItem

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

	if err == models.ErrNoData {
		// the item doesn't exist in the branch-items list
		cached_branch_item = &CachedBranchItem{
			ShBranchItem:       search_item,
			itemExistsInBranch: false,
			itemVisited:        false}
	} else if err == nil {
		cached_branch_item = &CachedBranchItem{
			ShBranchItem:       branch_item,
			itemExistsInBranch: true,
			itemVisited:        false}
	} else if err != nil {
		// TODO: handle the error properly
		// handle the case when the error is other type
	}

	cached_branch_item.itemVisited = false

	if !cached_branch_item.itemExistsInBranch {
		cached_branch_item.Quantity = float64(0)
	}

	seenItems[Pair_BranchItem{branch_id, item_id}] = cached_branch_item
	return cached_branch_item
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
func updateBranchItems(tnx *sql.Tx,
	affected_branch_items AffectedBranchItems,
	company_id int) error {

	for pair_branch_item, cached_item := range affected_branch_items {
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

func addTransactions(tnx *sql.Tx,
	request *sp.TransactionRequest,
	user_info *UserCompanyPermission,
) (
	affected_branch_items AffectedBranchItems,
	old_2_new map[int64]int64,
	err error,
) {
	affected_branch_items = make(AffectedBranchItems)
	old_2_new = make(map[int64]int64)

	company_id := user_info.CompanyId
	user_id := user_info.User.UserId

	for _, posted_trans := range request.Transactions {
		/**
		 * If the transaction already exists, that must mean the user didn't
		 * get acknowledgement when posting and is trying to re-post, so just send
		 * them the id.
		 */
		if prev_trans, err := Store.GetShTransactionByUUIDInTx(tnx, posted_trans.UUID); err == nil {
			old_2_new[posted_trans.TransId] = prev_trans.TransactionId
			continue
		} else if err != models.ErrNoData {
			return nil, nil, err
		}

		trans := new(models.ShTransaction)

		trans.CompanyId = company_id
		trans.UserId = user_id

		trans.TransactionId = posted_trans.TransId
		trans.BranchId = int(posted_trans.BranchId)
		trans.Date = posted_trans.DateTime
		trans.TransNote = posted_trans.TransNote
		trans.ClientUUID = posted_trans.UUID

		for _, _item := range posted_trans.TransactionItems {
			trans.TransItems = append(trans.TransItems,
				&models.ShTransactionItem{
					CompanyId:     company_id,
					TransType:     int(_item.TransType),
					ItemId:        int(_item.ItemId),
					OtherBranchId: int(_item.OtherBranchId),
					Quantity:      _item.Quantity,
					ItemNote:      _item.ItemNote,
				})
		}
		created, err := Store.CreateShTransactionInTx(tnx, trans)
		if err != nil {
			return nil, nil, err
		}

		old_2_new[posted_trans.TransId] = created.TransactionId

		for _, trans_item := range created.TransItems {
			branch_item := searchBranchItemInCache(tnx, affected_branch_items,
				&models.ShBranchItem{
					CompanyId: company_id, BranchId: int(posted_trans.BranchId),
					ItemId: trans_item.ItemId,
				})
			branch_item.itemVisited = true

			switch trans_item.TransType {
			case models.TRANS_TYPE_ADD_PURCHASED:
				branch_item.Quantity += trans_item.Quantity

			case models.TRANS_TYPE_SUB_CURRENT_BRANCH_SALE:
				branch_item.Quantity -= trans_item.Quantity

			// these 2 affect another branch
			// so, grab that branch and update it also
			case models.TRANS_TYPE_ADD_TRANSFER_FROM_OTHER,
				models.TRANS_TYPE_SUB_TRANSFER_TO_OTHER:

				other_branch_item := searchBranchItemInCache(tnx, affected_branch_items,
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

	return affected_branch_items, old_2_new, nil
}

func (s *SheketController) SyncTransaction(c context.Context, request *sp.TransactionRequest) (response *sp.TransactionResponse, err error) {
	defer trace("SyncTransaction")()

	user_info, err := GetUserWithCompanyPermission(request.CompanyAuth)
	if err != nil {
		return nil, err
	}

	tnx, err := Store.Begin()
	if err != nil {
		return nil, err
	}

	var old_2_new map[int64]int64
	var affected_branch_items AffectedBranchItems

	if affected_branch_items, old_2_new, err = addTransactions(tnx, request, user_info); err != nil {
		tnx.Rollback()
		return nil, err
	}

	// update items affected by the transactions
	if err = updateBranchItems(tnx, affected_branch_items, user_info.CompanyId); err != nil {
		tnx.Rollback()
		return nil, err
	}
	tnx.Commit()

	response = new(sp.TransactionResponse)
	for old_id, new_id := range old_2_new {
		response.UpdatedTransactionIds = append(response.UpdatedTransactionIds,
			&sp.TransactionResponse_UpdatedTransId{
				OldId: old_id,
				NewId: new_id,
			})
	}

	if err = fetchBranchItemsSinceRev(request, response, old_2_new, user_info.CompanyId); err != nil {
		return nil, err
	}

	if user_info.Permission.HasManagerAccess() {
		if err := fetchTransactionsSince(request, response, old_2_new, user_info.CompanyId); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func fetchTransactionsSince(
	request *sp.TransactionRequest,
	response *sp.TransactionResponse,
	old_2_new map[int64]int64,
	company_id int) error {

	transactions, err := Store.GetShTransactionSinceTransId(company_id, request.OldTransRev)
	if err != nil {
		return err
	}

	max_trans_id := request.OldTransRev

	for _, trans := range transactions {
		if trans.TransactionId > max_trans_id {
			max_trans_id = trans.TransactionId
		}

		/*
			// ignore currently added new transactions in the sync
			if newly_created_ids[trans.TransactionId] {
				continue
			}
		*/

		var transItems []*sp.Transaction_TransItem
		for _, _item := range trans.TransItems {
			transItems = append(transItems,
				&sp.Transaction_TransItem{
					TransType:     int32(_item.TransType),
					ItemId:        int32(_item.ItemId),
					OtherBranchId: int32(_item.OtherBranchId),
					Quantity:      _item.Quantity,
					ItemNote:      _item.ItemNote,
				})
		}

		response.Transactions = append(response.Transactions,
			&sp.TransactionResponse_SyncTransaction{
				UserId: int32(trans.UserId),
				Transaction: &sp.Transaction{
					TransId:          trans.TransactionId,
					UUID:             trans.ClientUUID,
					BranchId:         int32(trans.BranchId),
					DateTime:         trans.Date,
					TransNote:        trans.TransNote,
					TransactionItems: transItems,
				},
			})
	}

	response.NewTransRev = max_trans_id

	return nil
}

func fetchBranchItemsSinceRev(
	request *sp.TransactionRequest,
	response *sp.TransactionResponse,
	old_2_new map[int64]int64,
	company_id int) error {

	max_rev, new_branch_item_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH_ITEM,
			RevisionNumber: int(request.OldBranchItemRev),
		})

	if err != nil {
		return err
	}

	response.NewBranchItemRev = int64(max_rev)

	for _, branch_rev := range new_branch_item_revs {
		branch_id := branch_rev.EntityAffectedId
		item_id := branch_rev.AdditionalInfo

		branch_item, err := Store.GetBranchItem(branch_id, item_id)
		if err != nil {
			if err != models.ErrNoData {
				return err
			}
			continue
		}

		// TODO: add check to not revisit the same branch item
		// Also check the revision's ACTION to decide ACTION
		// Check if doing so messes up having the "visited" option
		response.BranchItems = append(response.BranchItems,
			&sp.TransactionResponse_SyncBranchItem{
				BranchItem: &sp.BranchItem{
					BranchId:      int32(branch_id),
					ItemId:        int32(item_id),
					Quantity:      branch_item.Quantity,
					ShelfLocation: branch_item.ItemLocation,
				},
			})
	}

	return nil
}
