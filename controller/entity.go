package controller

import (
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
	OldId2New_Categories map[int64]int64
	OldId2New_Items      map[int64]int64
	OldId2New_Branches   map[int64]int64

	NewlyCreatedCategoryIds map[int64]bool
	NewlyCreatedItemIds     map[int64]bool
	NewlyCreatedBranchIds   map[int64]bool
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
