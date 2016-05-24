package controller

import (
	"database/sql"
	"sheket/server/models"
	"fmt"
)

func applyEntityOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo) (*EntityResult, error) {
	result := &EntityResult{}

	result.OldId2New_Categories = make(map[int64]int64)
	result.OldId2New_Items = make(map[int64]int64)
	result.OldId2New_Branches = make(map[int64]int64)

	result.NewlyCreatedCategoryIds = make(map[int64]bool)
	result.NewlyCreatedItemIds = make(map[int64]bool)
	result.NewlyCreatedBranchIds = make(map[int64]bool)

	var err error
	result, err = applyCategoryOperations(tnx, posted_data, info, result)
	if err != nil {
		return nil, err
	}

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

func applyCategoryOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo, result *EntityResult) (*EntityResult, error) {
	if len(posted_data.CategoryFields) == 0 {
		return result, nil
	}

	// visit every category first, to see which ones are available
	if len(posted_data.CategoryIds[ACTION_CREATE]) > 0 {
		i := 0
		category_ids := make([]int64, len(posted_data.CategoryIds[ACTION_CREATE]))
		for id := range posted_data.CategoryIds[ACTION_CREATE] {
			category_ids[i] = id
			i++
		}
		// reset it to the last position
		i--

		/**
		 * Since some categories have their parent categories which still have not been created,
		 * it creates a dependency tree where a category can't be created until its parent
		 * is created.
		 *
		 * So, we do that by creating a stack of category ids to perform operations in. We pop
		 * an id and see if its dependency has been fulfilled. It it has, we go ahead and create
		 * the category, otherwise we return the category back to the stack and add its
		 * dependency(in this case its parent) to the stack to do its operation in the next round.
		 */

		for i >= 0 {
			id := category_ids[i]
			category, ok := posted_data.CategoryFields[id]
			if !ok {
				return nil, fmt.Errorf("category:%d doesn't have members defined", id)
			}
			// Check if the category has already been created in another sync round
			prev_category, err := Store.GetCategoryByUUIDInTx(tnx, category.ClientUUID)
			if err != nil {
				return nil, fmt.Errorf("error getting category with uuid %s", err.Error())
			} else if prev_category != nil {
				result.OldId2New_Categories[id] = prev_category.CategoryId
				result.NewlyCreatedCategoryIds[prev_category.CategoryId] = true
				// pop-off the stack for the next round
				i--
				continue
			}

			// if the parent has been added to the DataStore, update the id with the new id
			if new_parent_id, ok := result.OldId2New_Categories[category.ParentId]; ok {
				category.ParentId = new_parent_id
			} else if (category.ParentId != models.ROOT_CATEGORY_ID) &&
				category.ParentId < 0 {	// if we didn't add it and it is not root, add the parent first

				// TODO: figure out a better way to extend an array size, this is just a hack
				// it is a hack b/c we append an elem to make sure it has enough capacity, we then
				// add the parent_id to the to the "stack" to be used in the next round
				category_ids = append(category_ids, category.ParentId)
				category_ids[i+1] = category.ParentId
				// push the element to the stack
				i++
				continue
			}

			i--		// pop off the stack for the next round
			created_category, err := Store.CreateCategoryInTx(tnx, &category.ShCategory)
			if err != nil {
				return nil, fmt.Errorf("error creating category %s", err.Error())
			}

			result.OldId2New_Categories[id] = created_category.CategoryId
			result.NewlyCreatedCategoryIds[created_category.CategoryId] = true

			rev := &models.ShEntityRevision{
				CompanyId:        info.CompanyId,
				EntityType:       models.REV_ENTITY_CATEGORY,
				ActionType:       models.REV_ACTION_CREATE,
				EntityAffectedId: created_category.CategoryId,
				AdditionalInfo:   -1,
			}

			_, err = Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return nil, err
			}
		}
	}

	for category_id := range posted_data.CategoryIds[ACTION_UPDATE] {
		category, ok := posted_data.CategoryFields[category_id]
		if !ok {
			return nil, fmt.Errorf("category:%d doesn't have member fields defined", category_id)
		}
		prev_category, err := Store.GetCategoryByIdInTx(tnx, category_id)
		if err != nil {
			return nil, fmt.Errorf("error retriving category:%d '%s'", category_id, err.Error())
		}

		if category.SetFields[models.CATEGORY_JSON_NAME] {
			prev_category.Name = category.Name
		}
		if category.SetFields[models.CATEGORY_JSON_PARENT_ID] {
			prev_category.ParentId = category.ParentId
		}

		if _, err = Store.UpdateCategoryInTx(tnx, prev_category); err != nil {
			return nil, fmt.Errorf("error updating category:%d '%s'", category_id, err.Error())
		}
		rev := &models.ShEntityRevision{
			CompanyId:        info.CompanyId,
			EntityType:       models.REV_ENTITY_CATEGORY,
			ActionType:       models.REV_ACTION_UPDATE,
			EntityAffectedId: category.CategoryId,
			AdditionalInfo:   -1,
		}

		_, err = Store.AddEntityRevisionInTx(tnx, rev)
		if err != nil {
			return nil, err
		}
	}

	// TODO: implement deletion
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
			return nil, fmt.Errorf("error getting item with uuid %s", err.Error())
		} else if prev_item != nil {
			result.OldId2New_Items[old_item_id] = prev_item.ItemId
			result.NewlyCreatedItemIds[prev_item.ItemId] = true
			break
		}

		if new_category_id, ok := result.OldId2New_Categories[item.CategoryId]; ok {
			item.CategoryId = new_category_id
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

		if new_category_id, ok := result.OldId2New_Categories[item.CategoryId]; ok {
			item.CategoryId = new_category_id
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
		if item.SetFields[models.ITEM_JSON_CATEGORY_ID] {
			previous_item.CategoryId = item.CategoryId
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
