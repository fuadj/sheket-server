package controller

import (
	"fmt"
	"net/http"
	_ "net/http/httputil"
	"sheket/server/models"
	"github.com/gin-gonic/gin"
	_ "net/http/httputil"
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

	type_categories   = "category"
	type_items        = "item"
	type_branches     = "branch"
	type_branch_items = "branch_item"
	type_members      = "member"

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

func EntitySyncHandler(c *gin.Context) {
	defer trace("EntitySyncHandler")()

	/*
	d, err := httputil.DumpRequest(c.Request, true)
	if err == nil {
		fmt.Printf("Request %s\n", string(d))
	}
	*/

	info, err := GetIdentityInfo(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{ERROR_MSG:err.Error()})
		return
	}

	posted_data, err := parseEntityPost(c.Request.Body, parsers, info)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ERROR_MSG:err.Error()})
		return
	}

	tnx, err := Store.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:err.Error()})
		return
	}

	result, err := applyEntityOperations(tnx, posted_data, info)
	if err != nil {
		tnx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:err.Error()})
		return
	}
	tnx.Commit()

	response := make(map[string]interface{})
	response[JSON_KEY_COMPANY_ID] = info.CompanyId

	if err = syncNewEntities(response, result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:err.Error()})
		return
	}

	if err = syncModifiedEntities(response, posted_data, result, info); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ERROR_MSG:err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func syncNewEntities(sync_response map[string]interface{}, sync_result *EntityResult) error {
	if len(sync_result.OldId2New_Categories) > 0 {
		i := int64(0)
		updated_ids := make([]map[string]int64, len(sync_result.OldId2New_Categories))
		for old_id, new_id := range sync_result.OldId2New_Categories {
			updated_ids[i] = map[string]int64{
				KEY_JSON_ID_OLD: old_id,
				KEY_JSON_ID_NEW: new_id,
			}
			i++
		}

		sync_response[key_updated_category_ids] = updated_ids
	}

	if len(sync_result.OldId2New_Items) > 0 {
		i := int64(0)
		updated_ids := make([]map[string]int64, len(sync_result.OldId2New_Items))
		for old_id, new_id := range sync_result.OldId2New_Items {
			updated_ids[i] = map[string]int64{
				KEY_JSON_ID_OLD: old_id,
				KEY_JSON_ID_NEW: new_id,
			}
			i++
		}

		sync_response[key_updated_item_ids] = updated_ids
	}
	if len(sync_result.OldId2New_Branches) > 0 {
		i := int64(0)
		updated_ids := make([]map[string]int64, len(sync_result.OldId2New_Branches))
		for old_id, new_id := range sync_result.OldId2New_Branches {
			updated_ids[i] = map[string]int64{
				KEY_JSON_ID_OLD: old_id,
				KEY_JSON_ID_NEW: new_id,
			}
			i++
		}

		sync_response[key_updated_branch_ids] = updated_ids
	}
	return nil
}

func syncModifiedEntities(sync_response map[string]interface{},
	posted_data *EntitySyncData, sync_result *EntityResult,
	info *IdentityInfo) error {

	latest_category_rev, changed_categories, err := fetchChangedCategoriesSinceRev(info.CompanyId,
		posted_data.RevisionCategory, sync_result.NewlyCreatedCategoryIds)
	if err != nil {
		return err
	}
	sync_response[key_category_revision] = latest_category_rev
	if len(changed_categories) > 0 {
		sync_response[key_sync_categories] = changed_categories
	}

	latest_item_rev, changed_items, err := fetchChangedItemsSinceRev(info.CompanyId,
		posted_data.RevisionItem, sync_result.NewlyCreatedItemIds)
	if err != nil {
		return err
	}
	sync_response[key_item_revision] = latest_item_rev
	if len(changed_items) > 0 {
		sync_response[key_sync_items] = changed_items
	}

	latest_branch_rev, changed_branches, err := fetchChangedBranchesSinceRev(info.CompanyId,
		posted_data.RevisionBranch, sync_result.NewlyCreatedBranchIds)
	if err != nil {
		return err
	}
	sync_response[key_branch_revision] = latest_branch_rev
	if len(changed_branches) > 0 {
		sync_response[key_sync_branches] = changed_branches
	}

	if info.Permission.PermissionType <= models.PERMISSION_TYPE_BRANCH_MANAGER {
		max_member_rev, members, err := fetchChangedMemberSinceRev(info.CompanyId,
			posted_data.RevisionMember)
		if err != nil {
			return err
		}
		if len(members) > 0 {
			sync_response[key_sync_members] = members
		}
		sync_response[key_member_revision] = max_member_rev
	}
	return nil
}

func fetchChangedCategoriesSinceRev(company_id, last_category_rev int64, newly_created_category_ids map[int64]bool) (latest_rev int64,
	categories_since []map[string]interface{}, err error) {
	max_rev, changed_category_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_CATEGORY,
			RevisionNumber: last_category_rev,
		})
	if err != nil {
		return max_rev, nil, err
	}

	result := make([]map[string]interface{}, len(changed_category_revs))
	i := 0
	for _, category_rev := range changed_category_revs {
		category_id := category_rev.EntityAffectedId
		// the category was created in this sync "round", we already have it
		if newly_created_category_ids[category_id] {
			continue
		}
		category, err := Store.GetCategoryById(category_id)
		if err != nil {
			// TODO: differentiate deleted from error
			fmt.Printf("GetCategoryById error '%v'", err.Error())
			continue
		}

		// convert back to client root category id
		if category.ParentId == models.ROOT_CATEGORY_ID {
			category.ParentId = CLIENT_ROOT_CATEGORY_ID
		}

		result[i] = map[string]interface{}{
			models.CATEGORY_JSON_CATEGORY_ID: category.CategoryId,
			models.CATEGORY_JSON_NAME:        category.Name,
			models.CATEGORY_JSON_PARENT_ID:   category.ParentId,
			models.CATEGORY_JSON_UUID:        category.ClientUUID,
		}
		i++
	}
	return max_rev, result[:i], nil
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

		if item.CategoryId == models.ROOT_CATEGORY_ID {
			item.CategoryId = CLIENT_ROOT_CATEGORY_ID
		}

		result[i] = map[string]interface{}{
			models.ITEM_JSON_ITEM_ID:     item.ItemId,
			models.ITEM_JSON_UUID:        item.ClientUUID,
			models.ITEM_JSON_ITEM_NAME:   item.Name,
			models.ITEM_JSON_ITEM_CODE:   item.ItemCode,
			models.ITEM_JSON_CATEGORY_ID: item.CategoryId,

			models.ITEM_JSON_UNIT_OF_MEASUREMENT: item.UnitOfMeasurement,
			models.ITEM_JSON_HAS_DERIVED_UNIT:    item.HasDerivedUnit,
			models.ITEM_JSON_DERIVED_NAME:        item.DerivedName,
			models.ITEM_JSON_DERIVED_FACTOR:      item.DerivedFactor,
			models.ITEM_JSON_REORDER_LEVEL:       item.ReorderLevel,

			models.ITEM_JSON_MODEL_YEAR:   item.ModelYear,
			models.ITEM_JSON_PART_NUMBER:  item.PartNumber,
			models.ITEM_JSON_BAR_CODE:     item.BarCode,
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
