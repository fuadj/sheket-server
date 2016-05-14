package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sheket/server/models"
)

const (
	key_item_revision     = "item_rev"
	key_branch_revision   = "branch_rev"
	key_member_revision   = "member_rev"
	key_category_revision = "category_rev"

	key_types = "types"

	key_created = "create"
	key_updated = "update"
	key_deleted = "delete"

	key_fields = "fields"

	type_categories   = "categories"
	type_items        = "items"
	type_branches     = "branches"
	type_branch_items = "branch_items"
	type_members      = "members"

	// used in the response to hold the newly updated category ids
	key_updated_category_ids = "updated_category_ids"

	// used in the response json to hold the newly updated item ids
	key_updated_item_ids = "updated_item_ids"

	// used in the response json to hold the newly updated branch ids
	key_updated_branch_ids = "updated_branch_ids"

	// key of json holding any updated categories since last sync
	key_sync_categories = "sync_categories"

	// key of json holding any updated items since last sync
	key_sync_items = "sync_items"

	// key of json holding any updated branches since last sync
	key_sync_branches = "sync_branches"

	key_sync_members = "sync_members"
)

type EntityResult struct {
	OldId2New_Items    map[int64]int64
	OldId2New_Branches map[int64]int64

	NewlyCreatedItemIds   map[int64]bool
	NewlyCreatedBranchIds map[int64]bool
}

func EntitySyncHandler(w http.ResponseWriter, r *http.Request) {
	defer trace("EntitySyncHandler")()
	d, err := httputil.DumpRequest(r, true)
	if err == nil {
		fmt.Printf("Request %s\n", string(d))
	}

	info, err := GetIdentityInfo(r)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}

	posted_data, err := parseEntityPost(r.Body, parsers, info)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	tnx, err := Store.Begin()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "e0")
		return
	}

	result, err := applyEntityOperations(tnx, posted_data, info)
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, err.Error()+"e1")
		return
	}
	tnx.Commit()

	sync_result := make(map[string]interface{})
	sync_result[JSON_KEY_COMPANY_ID] = info.CompanyId

	// if there were newly added items, send to user updated ids
	if len(result.OldId2New_Items) > 0 {
		i := int64(0)
		updated_ids := make([]map[string]int64, len(result.OldId2New_Items))
		for old_id, new_id := range result.OldId2New_Items {
			updated_ids[i] = map[string]int64{
				KEY_JSON_ID_OLD: old_id,
				KEY_JSON_ID_NEW: new_id,
			}
			i++
		}

		sync_result[key_updated_item_ids] = updated_ids
	}
	if len(result.OldId2New_Branches) > 0 {
		i := int64(0)
		updated_ids := make([]map[string]int64, len(result.OldId2New_Branches))
		for old_id, new_id := range result.OldId2New_Branches {
			updated_ids[i] = map[string]int64{
				KEY_JSON_ID_OLD: old_id,
				KEY_JSON_ID_NEW: new_id,
			}
			i++
		}

		sync_result[key_updated_branch_ids] = updated_ids
	}

	latest_item_rev, changed_items, err := fetchChangedItemsSinceRev(info.CompanyId,
		posted_data.RevisionItem, result.NewlyCreatedItemIds)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error()+"e2")
		return
	}
	sync_result[key_item_revision] = latest_item_rev
	if len(changed_items) > 0 {
		sync_result[key_sync_items] = changed_items
	}
	latest_branch_rev, changed_branches, err := fetchChangedBranchesSinceRev(info.CompanyId,
		posted_data.RevisionBranch, result.NewlyCreatedBranchIds)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error()+"e3")
		return
	}
	sync_result[key_branch_revision] = latest_branch_rev
	if len(changed_branches) > 0 {
		sync_result[key_sync_branches] = changed_branches
	}

	if info.Permission.PermissionType <= models.PERMISSION_TYPE_BRANCH_MANAGER {
		max_member_rev, members, err := fetchChangedMemberSinceRev(info.CompanyId,
			posted_data.RevisionMember)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError)
			return
		}
		if len(members) > 0 {
			sync_result[key_sync_members] = members
		}
		sync_result[key_member_revision] = max_member_rev
	}

	b, err := json.MarshalIndent(sync_result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "e4")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	s := string(b)
	fmt.Printf("Entity Sync response size:(%d)bytes\n%s\n\n\n", len(s), s)
}

func applyEntityOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo) (*EntityResult, error) {
	result := &EntityResult{}
	result.OldId2New_Items = make(map[int64]int64)
	result.OldId2New_Branches = make(map[int64]int64)

	result.NewlyCreatedItemIds = make(map[int64]bool)
	result.NewlyCreatedBranchIds = make(map[int64]bool)

	var err error
	result, err = applyItemOperations(tnx, posted_data, info, result)
	if err != nil {
		return nil, err
	}

	result, err = applyBranchOperations(tnx, posted_data, info, result)
	if err != nil {
		return nil, err
	}

	result, err = applyBranchItemOperations(tnx, posted_data, info, result)
	if err != nil {
		return nil, err
	}

	if info.Permission.PermissionType <= models.PERMISSION_TYPE_MANAGER {
		result, err = applyMemberOperations(tnx, posted_data, info, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func applyItemOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo, result *EntityResult) (*EntityResult, error) {
	// short-cut
	if len(posted_data.ItemFields) == 0 {
		return result, nil
	}

	// for the newly created items, get a globally after adding them to datastore
	for old_item_id := range posted_data.ItemIds[ACTION_CREATE] {
		item, ok := posted_data.ItemFields[old_item_id]
		if !ok {
			return nil, fmt.Errorf("item:%d doesn't have members defined", old_item_id)
		}
		prev_item, err := Store.GetItemByUUIDInTx(tnx, item.ClientUUID)
		if err != nil {
			return nil, fmt.Errorf("error getting item with uuid", err.Error())
		} else if prev_item != nil {
			result.OldId2New_Items[old_item_id] = prev_item.ItemId
			result.NewlyCreatedItemIds[prev_item.ItemId] = true
			break
		}

		created_item, err := Store.CreateItemInTx(tnx, &item.ShItem)
		if err != nil {
			return nil, fmt.Errorf("error creating item %s", err.Error())
		}
		result.OldId2New_Items[old_item_id] = created_item.ItemId
		result.NewlyCreatedItemIds[created_item.ItemId] = true

		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_ITEM,
			ActionType:       models.REV_ACTION_CREATE,
			EntityAffectedId: created_item.ItemId,
			AdditionalInfo:   -1,
		}

		_, err = Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}
	// for items being updated, see which members the user uploaded,
	// and update those
	for item_id := range posted_data.ItemIds[ACTION_UPDATE] {
		item, ok := posted_data.ItemFields[item_id]
		if !ok {
			return nil, fmt.Errorf("item:%d doesn't have members defined", item_id)
		}
		previous_item, err := Store.GetItemByIdInTx(tnx, item_id)
		if err != nil {
			return nil, fmt.Errorf("error retriving item:%d '%s'", item_id, err.Error())
		}

		if item.SetFields[models.ITEM_JSON_MODEL_YEAR] {
			previous_item.ModelYear = item.ModelYear
		}
		if item.SetFields[models.ITEM_JSON_PART_NUMBER] {
			previous_item.PartNumber = item.PartNumber
		}
		if item.SetFields[models.ITEM_JSON_BAR_CODE] {
			previous_item.BarCode = item.BarCode
		}
		if item.SetFields[models.ITEM_JSON_MANUAL_CODE] {
			previous_item.ManualCode = item.ManualCode
		}
		if item.SetFields[models.ITEM_JSON_HAS_BAR_CODE] {
			previous_item.HasBarCode = item.HasBarCode
		}

		if _, err = Store.UpdateItemInTx(tnx, previous_item); err != nil {
			return nil, fmt.Errorf("error updating item:%d '%v'", item_id, err.Error())
		}

		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_ITEM,
			ActionType:       models.REV_ACTION_UPDATE,
			EntityAffectedId: item_id,
			AdditionalInfo:   -1,
		}

		_, err = Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}

	// TODO: delete item not yet implemented

	return result, nil
}

func applyBranchOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo, result *EntityResult) (*EntityResult, error) {
	// short-cut
	if len(posted_data.BranchFields) == 0 {
		return result, nil
	}

	for old_branch_id := range posted_data.BranchIds[ACTION_CREATE] {
		branch, ok := posted_data.BranchFields[old_branch_id]
		if !ok {
			return nil, fmt.Errorf("branch:%d doesn't have members defined")
		}
		prev_branch, err := Store.GetBranchByUUIDInTx(tnx, branch.ClientUUID)
		if err != nil {
			return nil, err
		} else if prev_branch != nil {
			result.OldId2New_Branches[old_branch_id] = prev_branch.BranchId
			result.NewlyCreatedBranchIds[prev_branch.BranchId] = true
			continue
		}

		created_branch, err := Store.CreateBranchInTx(tnx, &branch.ShBranch)
		if err != nil {
			return nil, fmt.Errorf("error creating branch %s", err.Error())
		}
		result.OldId2New_Branches[old_branch_id] = created_branch.BranchId
		result.NewlyCreatedBranchIds[created_branch.BranchId] = true

		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_BRANCH,
			ActionType:       models.REV_ACTION_CREATE,
			EntityAffectedId: created_branch.BranchId,
			AdditionalInfo:   -1,
		}

		_, err = Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}
	for branch_id := range posted_data.BranchIds[ACTION_UPDATE] {
		branch, ok := posted_data.BranchFields[branch_id]
		if !ok {
			return nil, fmt.Errorf("branch:%d doesn't have members defined")
		}
		previous_branch, err := Store.GetBranchByIdInTx(tnx, branch_id)
		if err != nil {
			return nil, fmt.Errorf("error retriving branch:%d '%s'", branch_id, err.Error())
		}

		if branch.SetFields[models.BRANCH_JSON_NAME] {
			previous_branch.Name = branch.Name
		}
		if branch.SetFields[models.BRANCH_JSON_LOCATION] {
			previous_branch.Location = branch.Location
		}

		if _, err = Store.UpdateBranchInTx(tnx, previous_branch); err != nil {
			return nil, fmt.Errorf("error updating branch:%d '%v'", branch_id, err.Error())
		}

		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_BRANCH,
			ActionType:       models.REV_ACTION_UPDATE,
			EntityAffectedId: branch_id,
			AdditionalInfo:   -1,
		}

		_, err = Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}
	// TODO: delete branch not yet implemented
	return result, nil
}

func applyBranchItemOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo, result *EntityResult) (*EntityResult, error) {
	// short-cut
	if len(posted_data.Branch_ItemFields) == 0 {
		return result, nil
	}
	for pair_branch_item := range posted_data.Branch_ItemIds[ACTION_CREATE] {
		branch_item, ok := posted_data.Branch_ItemFields[pair_branch_item]
		if !ok {
			return nil, fmt.Errorf("branchItem:(%d,%d) doesn't have members defined",
				pair_branch_item.BranchId, pair_branch_item.ItemId)
		}
		branch_id, item_id := pair_branch_item.BranchId, pair_branch_item.ItemId

		// if the id's of the branch item were the locally user generated(yet to be replaced with
		// server generated global ids), then use the server's
		if new_branch_id, ok := result.OldId2New_Branches[branch_id]; ok {
			branch_item.BranchId = new_branch_id
		}
		if new_item_id, ok := result.OldId2New_Items[item_id]; ok {
			branch_item.ItemId = new_item_id
		}

		branch_item.Quantity = 0 // only transactions affect quantity, not directly
		if _, err := Store.AddItemToBranchInTx(tnx, &branch_item.ShBranchItem); err != nil {
			return nil, fmt.Errorf("error adding item:%d to branch:%d '%s'",
				branch_item.ItemId, branch_item.BranchId, err.Error())
		}

		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_BRANCH_ITEM,
			ActionType:       models.REV_ACTION_CREATE,
			EntityAffectedId: branch_id,
			AdditionalInfo:   item_id,
		}

		_, err := Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}
	for pair_branch_item := range posted_data.Branch_ItemIds[ACTION_UPDATE] {
		posted_branch_item, ok := posted_data.Branch_ItemFields[pair_branch_item]
		if !ok {
			return nil, fmt.Errorf("branchItem:(%d,%d) doesn't have members defined",
				pair_branch_item.BranchId, pair_branch_item.ItemId)
		}
		branch_id, item_id := pair_branch_item.BranchId, pair_branch_item.ItemId

		// if the id's of the branch item were the locally user generated(yet to be replaced with
		// server generated global ids), then use the server's
		if new_branch_id, ok := result.OldId2New_Branches[branch_id]; ok {
			posted_branch_item.BranchId = new_branch_id
		}
		if new_item_id, ok := result.OldId2New_Items[item_id]; ok {
			posted_branch_item.ItemId = new_item_id
		}

		previous_item, err := Store.GetBranchItemInTx(tnx, branch_id, item_id)
		if err != nil {
			return nil, fmt.Errorf("error retriving branchItem:(%d,%d) '%s'", branch_id, item_id, err.Error())
		}

		if posted_branch_item.SetFields[models.BRANCH_ITEM_JSON_ITEM_LOCATION] {
			previous_item.ItemLocation = posted_branch_item.ItemLocation
		}

		if _, err = Store.UpdateBranchItemInTx(tnx, previous_item); err != nil {
			return nil, fmt.Errorf("error updating branchItem:(%d,%d) '%v'",
				branch_id, item_id, err.Error())
		}

		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_BRANCH_ITEM,
			ActionType:       models.REV_ACTION_UPDATE,
			EntityAffectedId: branch_id,
			AdditionalInfo:   item_id,
		}

		_, err = Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}

	// TODO: delete branch item not yet implemented
	return result, nil
}

func applyMemberOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo, result *EntityResult) (*EntityResult, error) {
	// short-cut
	if len(posted_data.MemberFields) == 0 {
		return result, nil
	}

	// TODO: we don't implement create here since it is directly online
	// and not on sync time
	for member_id := range posted_data.MemberIds[ACTION_UPDATE] {
		member, ok := posted_data.MemberFields[member_id]
		if !ok {
			return nil, fmt.Errorf("member:%d doesn't have fields defined", member_id)
		}

		p := &models.UserPermission{}
		p.CompanyId = info.CompanyId
		p.UserId = member.UserId
		p.EncodedPermission = member.EncodedPermission

		if _, err := Store.SetUserPermission(p); err != nil {
			return nil, fmt.Errorf("error updating member:%d permission '%v'", member_id, err)
		}

		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_MEMBERS,
			ActionType:       models.REV_ACTION_UPDATE,
			EntityAffectedId: member_id,
			AdditionalInfo:   -1,
		}

		_, err := Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func fetchChangedItemsSinceRev(company_id, item_rev int64, newly_created_item_ids map[int64]bool) (latest_rev int64,
	items_since []map[string]interface{}, err error) {

	max_rev, changed_item_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_ITEM,
			RevisionNumber: item_rev,
		})
	if err != nil {
		return max_rev, nil, err
	}

	result := make([]map[string]interface{}, len(changed_item_revs))
	i := 0
	for _, item_rev := range changed_item_revs {
		item_id := item_rev.EntityAffectedId
		if newly_created_item_ids[item_id] {
			continue
		}

		item, err := Store.GetItemById(item_id)
		if err != nil {
			fmt.Printf("GetItemById error '%v'", err.Error())
			continue
		}

		result[i] = map[string]interface{}{
			models.ITEM_JSON_ITEM_ID:      item.ItemId,
			models.ITEM_JSON_UUID:         item.ClientUUID,
			models.ITEM_JSON_ITEM_NAME:    item.Name,
			models.ITEM_JSON_MODEL_YEAR:   item.ModelYear,
			models.ITEM_JSON_PART_NUMBER:  item.PartNumber,
			models.ITEM_JSON_BAR_CODE:     item.BarCode,
			models.ITEM_JSON_MANUAL_CODE:  item.ManualCode,
			models.ITEM_JSON_HAS_BAR_CODE: item.HasBarCode,
		}
		i++
	}
	return max_rev, result[:i], nil
}

func fetchChangedBranchesSinceRev(company_id, branch_rev int64, newly_created_branch_ids map[int64]bool) (latest_rev int64,
	branches_since []map[string]interface{}, err error) {

	max_rev, new_branch_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH,
			RevisionNumber: branch_rev,
		})
	if err != nil {
		return max_rev, nil, err
	}
	result := make([]map[string]interface{}, len(new_branch_revs))

	i := 0
	for _, item_rev := range new_branch_revs {
		branch_id := item_rev.EntityAffectedId
		if newly_created_branch_ids[branch_id] {
			continue
		}

		branch, err := Store.GetBranchById(branch_id)
		if err != nil {
			// if a branch has been deleted in future revisions, we won't see it
			// so skip any error valued branches, and only return to the user those
			// that are correctly fetched
			fmt.Printf("GetBranchById Error '%v'", err.Error())
			continue
		}

		result[i] = map[string]interface{}{
			models.BRANCH_JSON_BRANCH_ID: branch.BranchId,
			models.BRANCH_JSON_UUID:      branch.ClientUUID,
			models.BRANCH_JSON_NAME:      branch.Name,
			models.BRANCH_JSON_LOCATION:  branch.Location,
		}
		i++
	}
	return max_rev, result[:i], nil
}

func fetchChangedMemberSinceRev(company_id, member_rev int64) (latest_rev int64,
	members_since []map[string]interface{}, err error) {

	max_rev, changed_member_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_MEMBERS,
			RevisionNumber: member_rev,
		})
	if err != nil {
		return max_rev, nil, err
	}

	result := make([]map[string]interface{}, len(changed_member_revs))
	i := 0
	for _, rev := range changed_member_revs {
		member_id := rev.EntityAffectedId

		user, err := Store.FindUserById(member_id)
		if err != nil {
			fmt.Printf("fetch changes FindUserById error '%v'", err)
			continue
		}

		permission, err := Store.GetUserPermission(user, company_id)
		if err != nil {
			fmt.Printf("fetch changes GetUserPermission error '%v'", err)
			continue
		}

		result[i] = map[string]interface{}{
			models.PERMISSION_JSON_MEMBER_ID:         member_id,
			models.PERMISSION_JSON_MEMBER_PERMISSION: permission.EncodedPermission,
			JSON_KEY_USERNAME:                        user.Username,
		}

		i++
	}

	return max_rev, result[:i], nil
}
