package controller

import (
	"container/list"
	"database/sql"
	"fmt"
	"sheket/server/models"
	sp "sheket/server/sheketproto"
)

type _ID_TYPE int64

const (
	_TYPE_ITEM _ID_TYPE = 0 + iota
	_TYPE_BRANCH
	_TYPE_CATEGORY
)

type OLD_ID_2_NEW map[_ID_TYPE]map[int64]int64

func new_Old_2_New() OLD_ID_2_NEW {
	old_2_new := make(OLD_ID_2_NEW)
	old_2_new[_TYPE_ITEM] = make(map[int64]int64)
	old_2_new[_TYPE_BRANCH] = make(map[int64]int64)
	old_2_new[_TYPE_CATEGORY] = make(map[int64]int64)
	return old_2_new
}

func (old_2_new OLD_ID_2_NEW) getType(id_type _ID_TYPE) map[int64]int64 {
	return old_2_new[id_type]
}

func applyEntityOperations(tnx *sql.Tx,
	request *sp.EntityRequest,
	response *sp.EntityResponse,
	user_info *UserCompanyPermission) (old_2_new OLD_ID_2_NEW, err error) {

	old_2_new = new_Old_2_New()

	company_id := user_info.CompanyId

	var err error
	if err = applyCategoryOperations(tnx, request.Categories, old_2_new, company_id); err != nil {
		return nil, err
	}

	if err = applyItemOperations(tnx, request.Items, old_2_new, company_id); err != nil {
		return nil, err
	}

	if err = applyBranchOperations(tnx, request.Branches, old_2_new, company_id); err != nil {
		return nil, err
	}

	if err = applyBranchItemOperations(tnx, request.BranchItems, old_2_new, company_id); err != nil {
		return nil, err
	}

	if err = applyBranchCategoryOperations(tnx, request.BranchCategories, old_2_new, company_id); err != nil {
		return nil, err
	}

	if user_info.Permission.PermissionType <= models.PERMISSION_TYPE_MANAGER {
		if err = applyEmployeeOperations(tnx, request.Employees, company_id); err != nil {
			return nil, err
		}
	}


	return old_2_new, nil
}

func _to_sh_category(sp_category *sp.Category) *models.ShCategory {
	m_category := new(models.ShCategory)

	m_category.ClientUUID = sp_category.UUID
	m_category.Name = sp_category.Name
	m_category.CategoryId = sp_category.CategoryId
	m_category.ParentId = sp_category.ParentId

	return m_category
}

/**
 * Since some categories have their parent categories which still have not been created,
 * we need to create a dependency tree where a category can't be created until its parent
 * is created.
 *
 * We solve that by creating a stack of categories. We pop a category and see if
 * its dependency has been fulfilled. It it has, we go ahead and create it.
 * Otherwise we push back the category to the stack and add its
 * dependency(its parent) on top of it so its parent can be added first.
 */
func insertCreatedCategories(tnx *sql.Tx,
	posted_categories []*sp.EntityRequest_RequestCategory,
	old_2_new OLD_ID_2_NEW,
	company_id int64) error {

	category_stack := list.New()
	categories := make(map[int64]*models.ShCategory)

	for _, _category := range posted_categories {
		if _category.Action == sp.EntityRequest_CREATE {
			category := _to_sh_category(_category.Category)
			category.CompanyId = company_id

			category.ParentId = To_Server_Category_Id(category.ParentId)

			categories[category.CategoryId] = category
			category_stack.PushBack(category.CategoryId)
		}
	}

	created_categories := make(map[int64]bool)

	for category_stack.Len() > 0 {
		category_id, _ := category_stack.Back().Value.(int64)
		category := categories[category_id]

		if created_categories[category_id] {
			// pop-off the stack
			category_stack.Remove(category_stack.Back())
			continue
		}

		// Check if the category already exists
		if prev_category, err := Store.GetCategoryByUUIDInTx(tnx, category.ClientUUID); err == nil {
			old_2_new.getType(_TYPE_CATEGORY)[category.CategoryId] = prev_category.CategoryId

			// pop-off the stack
			category_stack.Remove(category_stack.Back())

			continue
		} else if err != models.ErrNoData {
			return fmt.Errorf("error getting category with uuid %s", err.Error())
		}

		// if the parent has been added and it has an "un-synced" id, update the id
		if new_parent_id, ok := old_2_new.getType(_TYPE_CATEGORY)[category.ParentId]; ok {
			category.ParentId = new_parent_id
		} else if (category.ParentId < 0) &&
			(category.ParentId != models.SERVER_ROOT_CATEGORY_ID) {

			// if the parent hasn't been created and it is not ROOT,
			// add the parent to the top of the stack so it is visited next
			category_stack.PushBack(category.ParentId)
			continue
		}

		var new_category_id int64
		if category, err := Store.CreateCategoryInTx(tnx, category); err == nil {
			new_category_id = category.CategoryId
		} else {
			return fmt.Errorf("error creating category %s", err.Error())
		}

		rev := &models.ShEntityRevision{
			CompanyId:        company_id,
			EntityType:       models.REV_ENTITY_CATEGORY,
			ActionType:       models.REV_ACTION_CREATE,
			EntityAffectedId: new_category_id,
			AdditionalInfo:   -1,
		}

		if _, err := Store.AddEntityRevisionInTx(tnx, rev); err != nil {
			return err
		}

		created_categories[category_id] = true

		old_2_new.getType(_TYPE_CATEGORY)[category.CategoryId] = new_category_id

		// pop-off the stack
		category_stack.Remove(category_stack.Back())
	}

	return nil
}

func applyCategoryOperations(tnx *sql.Tx,
	posted_categories []*sp.EntityRequest_RequestCategory,
	old_2_new OLD_ID_2_NEW,
	company_id int64) error {

	if err := insertCreatedCategories(tnx, posted_categories, old_2_new, company_id); err != nil {
		return err
	}

	for _, _p_category := range posted_categories {
		category := _to_sh_category(_p_category.Category)
		category.CompanyId = company_id

		category.ParentId = To_Server_Category_Id(category.ParentId)

		switch _p_category.Action {
		case sp.EntityRequest_UPDATE:
			category_to_update, err := Store.GetCategoryByIdInTx(tnx, category.CategoryId)
			if err != nil {
				return fmt.Errorf("error retriving category:%d '%s'", category.CategoryId, err.Error())
			}

			category_to_update.Name = category.Name
			category_to_update.ParentId = category.ParentId

			if _, err = Store.UpdateCategoryInTx(tnx, category_to_update); err != nil {
				return fmt.Errorf("error updating category:%d '%s'", category.CategoryId, err.Error())
			}
			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_CATEGORY,
				ActionType:       models.REV_ACTION_UPDATE,
				EntityAffectedId: category.CategoryId,
				AdditionalInfo:   -1,
			}

			_, err = Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}

		case sp.EntityRequest_DELETE:
			if _, err := Store.GetCategoryByIdInTx(tnx, category.CategoryId); err != nil {
				if err != models.ErrNoData {
					continue
				} else {
					return fmt.Errorf("error retriving category:%d '%s'", category.CategoryId, err.Error())
				}
			}

			if err := Store.DeleteCategoryInTx(tnx, category.CategoryId); err != nil {
				return fmt.Errorf("error deleting category: %d '%s'", category.CategoryId, err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_CATEGORY,
				ActionType:       models.REV_ACTION_DELETE,
				EntityAffectedId: category.CategoryId,
				AdditionalInfo:   -1,
			}

			if _, err := Store.AddEntityRevisionInTx(tnx, rev); err != nil {
				return err
			}
		}
	}

	return nil
}

// converts from *sheket_proto.Item ==>> *models.ShItem
func _to_sh_item(sp_item *sp.Item) *models.ShItem {
	m_item := new(models.ShItem)

	m_item.ItemId = sp_item.ItemId
	m_item.ClientUUID = sp_item.UUID
	m_item.CategoryId = sp_item.CategoryId
	m_item.Name = sp_item.Name
	m_item.ItemCode = sp_item.Code
	m_item.UnitOfMeasurement = sp_item.UnitOfMeasurement
	m_item.HasDerivedUnit = sp_item.HasDerivedUnit
	m_item.DerivedName = sp_item.DerivedName
	m_item.DerivedFactor = sp_item.DerivedFactor
	m_item.StatusFlag = sp_item.StatusFlag

	return m_item
}

func applyItemOperations(tnx *sql.Tx,
	posted_items []*sp.EntityRequest_RequestItem,
	old_2_new OLD_ID_2_NEW,
	company_id int64) error {

	for _, _p_item := range posted_items {
		item := _to_sh_item(_p_item.Item)
		item.CompanyId = company_id
		item.CategoryId = To_Server_Category_Id(item.CategoryId)

		switch _p_item.Action {
		case sp.EntityRequest_CREATE:
			// check if it already exists
			if prev_item, err := Store.GetItemByUUIDInTx(tnx, _p_item.Item.UUID); err == nil {
				old_2_new.getType(_TYPE_ITEM)[_p_item.Item.ItemId] = prev_item.ItemId

				continue
			} else if err != models.ErrNoData {
				return err
			}

			if new_category_id, ok := old_2_new.getType(_TYPE_CATEGORY)[item.CategoryId]; ok {
				item.CategoryId = new_category_id
			}

			created_item, err := Store.CreateItemInTx(tnx, item)
			if err != nil {
				return fmt.Errorf("error creating item %s", err.Error())
			}
			old_2_new.getType(_TYPE_ITEM)[_p_item.Item.ItemId] = created_item.ItemId

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_ITEM,
				ActionType:       models.REV_ACTION_CREATE,
				EntityAffectedId: created_item.ItemId,
				AdditionalInfo:   -1,
			}

			if _, err = Store.AddEntityRevisionInTx(tnx, rev); err != nil {
				return err
			}

		case sp.EntityRequest_UPDATE:
			if new_category_id, ok := old_2_new.getType(_TYPE_CATEGORY)[item.CategoryId]; ok {
				item.CategoryId = new_category_id
			}

			item_to_update, err := Store.GetItemByIdInTx(tnx, item.ItemId)
			if err != nil {
				return fmt.Errorf("error retriving item:%d '%s'", item.ItemId, err.Error())
			}

			item_to_update.CategoryId = item.CategoryId
			item_to_update.Name = item.Name
			item_to_update.ItemCode = item.ItemCode

			item_to_update.UnitOfMeasurement = item.UnitOfMeasurement
			item_to_update.HasDerivedUnit = item.HasDerivedUnit
			item_to_update.DerivedName = item.DerivedName
			item_to_update.DerivedFactor = item.DerivedFactor
			item_to_update.ReorderLevel = item.ReorderLevel

			item_to_update.ModelYear = item.ModelYear
			item_to_update.PartNumber = item.PartNumber
			item_to_update.BarCode = item.BarCode
			item_to_update.HasBarCode = item.HasBarCode
			item_to_update.StatusFlag = item.StatusFlag

			if _, err = Store.UpdateItemInTx(tnx, item_to_update); err != nil {
				return fmt.Errorf("error updating item:%d '%v'", item.ItemId, err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_ITEM,
				ActionType:       models.REV_ACTION_UPDATE,
				EntityAffectedId: item.ItemId,
				AdditionalInfo:   -1,
			}

			_, err = Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}
		case sp.EntityRequest_DELETE:
			// TODO: item delete not implemented yet
		}
	}

	return nil
}

func _to_sh_branch(sp_branch *sp.Branch) *models.ShBranch {
	m_branch := new(models.ShBranch)

	m_branch.BranchId = sp_branch.BranchId
	m_branch.ClientUUID = sp_branch.UUID
	m_branch.Name = sp_branch.Name
	m_branch.StatusFlag = sp_branch.StatusFlag

	return m_branch
}

func applyBranchOperations(tnx *sql.Tx,
	posted_branches []*sp.EntityRequest_RequestBranch,
	old_2_new OLD_ID_2_NEW,
	company_id int64) error {

	for _, _p_branch := range posted_branches {
		branch := _to_sh_branch(_p_branch.Branch)
		branch.CompanyId = company_id

		switch _p_branch.Action {
		case sp.EntityRequest_CREATE:
			if prev_branch, err := Store.GetBranchByUUIDInTx(tnx, branch.ClientUUID); err == nil {
				old_2_new.getType(_TYPE_BRANCH)[branch.BranchId] = prev_branch.BranchId
				continue
			} else if err != models.ErrNoData {
				return err
			}

			created_branch, err := Store.CreateBranchInTx(tnx, branch)
			if err != nil {
				return fmt.Errorf("error creating branch %s", err.Error())
			}
			old_2_new.getType(_TYPE_BRANCH)[branch.BranchId] = created_branch.BranchId

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_BRANCH,
				ActionType:       models.REV_ACTION_CREATE,
				EntityAffectedId: created_branch.BranchId,
				AdditionalInfo:   -1,
			}

			_, err = Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}
		case sp.EntityRequest_UPDATE:
			branch_to_update, err := Store.GetBranchByIdInTx(tnx, branch.BranchId)
			if err != nil {
				return fmt.Errorf("error retriving branch:%d '%s'", branch.BranchId, err.Error())
			}

			branch_to_update.Name = branch.Name
			branch_to_update.Location = branch.Location
			branch_to_update.StatusFlag = branch.StatusFlag

			if _, err = Store.UpdateBranchInTx(tnx, branch_to_update); err != nil {
				return fmt.Errorf("error updating branch:%d '%v'", branch.BranchId, err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_BRANCH,
				ActionType:       models.REV_ACTION_UPDATE,
				EntityAffectedId: branch.BranchId,
				AdditionalInfo:   -1,
			}

			_, err = Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}
		case sp.EntityRequest_DELETE:
			// TODO: delete branch not yet implemented
		}
	}

	return nil
}

func _to_sh_branch_item(sp_branch_item *sp.BranchItem) *models.ShBranchItem {
	m_br_item := new(models.ShBranchItem)

	m_br_item.ItemId = sp_branch_item.ItemId
	m_br_item.BranchId = sp_branch_item.BranchId
	m_br_item.ItemLocation = sp_branch_item.ShelfLocation
	m_br_item.Quantity = sp_branch_item.Quantity

	return m_br_item
}

func applyBranchItemOperations(tnx *sql.Tx,
	posted_branch_items []*sp.EntityRequest_RequestBranchItem,
	old_2_new OLD_ID_2_NEW,
	company_id int64) error {

	for _, _p_branch_item := range posted_branch_items {
		b_item := _to_sh_branch_item(_p_branch_item.BranchItem)
		b_item.CompanyId = company_id

		switch _p_branch_item.Action {
		case sp.EntityRequest_CREATE:
			branch_id, item_id := b_item.BranchId, b_item.ItemId

			// if the id's of the branch item were the locally user generated(yet to be replaced with
			// server generated global ids), then use the server's
			if new_branch_id, ok := old_2_new.getType(_TYPE_BRANCH)[branch_id]; ok {
				b_item.BranchId = new_branch_id
			}
			if new_item_id, ok := old_2_new.getType(_TYPE_ITEM)[item_id]; ok {
				b_item.ItemId = new_item_id
			}

			b_item.Quantity = 0 // Start at 0 when adding it, let the transaction update it.
			if _, err := Store.AddItemToBranchInTx(tnx, b_item); err != nil {
				return fmt.Errorf("error adding item:%d to branch:%d '%s'",
					b_item.ItemId, b_item.BranchId, err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_BRANCH_ITEM,
				ActionType:       models.REV_ACTION_CREATE,
				EntityAffectedId: b_item.BranchId,
				AdditionalInfo:   b_item.ItemId,
			}

			_, err := Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}
		case sp.EntityRequest_UPDATE:
			branch_id, item_id := b_item.BranchId, b_item.ItemId

			// if the id's of the branch item were the locally user generated(yet to be replaced with
			// server generated global ids), then use the server's
			if new_branch_id, ok := old_2_new.getType(_TYPE_BRANCH)[branch_id]; ok {
				b_item.BranchId = new_branch_id
			}
			if new_item_id, ok := old_2_new.getType(_TYPE_ITEM)[item_id]; ok {
				b_item.ItemId = new_item_id
			}

			previous_branch_item, err := Store.GetBranchItemInTx(tnx, branch_id, item_id)
			if err != nil {
				return fmt.Errorf("error retriving branchItem:(%d,%d) '%s'", branch_id, item_id, err.Error())
			}

			// quantity isn't directly updatable, it is only affected through a transaction.
			// so only update the item location
			previous_branch_item.ItemLocation = b_item.ItemLocation

			if _, err = Store.UpdateBranchItemInTx(tnx, previous_branch_item); err != nil {
				return fmt.Errorf("error updating branchItem:(%d,%d) '%v'",
					branch_id, item_id, err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_BRANCH_ITEM,
				ActionType:       models.REV_ACTION_UPDATE,
				EntityAffectedId: branch_id,
				AdditionalInfo:   item_id,
			}

			_, err = Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}

		case sp.EntityRequest_DELETE:
			// TODO: delete branch item not yet implemented
		}
	}

	return nil
}

func _to_sh_user_permission(sp_employee *sp.Employee) *models.UserPermission {
	m_user_permission := new(models.UserPermission)

	m_user_permission.UserId = sp_employee.EmployeeId
	m_user_permission.EncodedPermission = sp_employee.Permission

	return m_user_permission
}

func applyEmployeeOperations(tnx *sql.Tx,
	posted_employees []*sp.EntityRequest_RequestEmployee,
	company_id int64) error {

	for _, _p_employee := range posted_employees {
		employee := _to_sh_user_permission(_p_employee.Employee)
		employee.CompanyId = company_id

		switch _p_employee.Action {
		case sp.EntityRequest_CREATE:
		// TODO: we don't implement create here b/c adding an employee is done directly online
		// NOT on sync time
		case sp.EntityRequest_UPDATE:

			if _, err := Store.SetUserPermission(employee); err != nil {
				return fmt.Errorf("error updating employee:%d permission '%v'",
					employee.UserId, err)
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_MEMBERS,
				ActionType:       models.REV_ACTION_UPDATE,
				EntityAffectedId: employee.UserId,
				AdditionalInfo:   -1,
			}

			_, err := Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}
		case sp.EntityRequest_DELETE:
			user, err := Store.FindUserById(employee.UserId)
			if err != nil {
				return fmt.Errorf("delete employee; error finding employee '%s'", err.Error())
			}

			_, err = Store.GetUserPermission(user, company_id)
			if err != nil && err != models.ErrNoData {
				return fmt.Errorf("delete employee; error finding employee:\"%s\" pemissions in company: %s",
					user.Username, err.Error())
			}

			err = Store.RemoveUserFromCompanyInTx(tnx, user.UserId, company_id)
			if err != nil {
				return fmt.Errorf("delete employee; delete error '%s'", err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_MEMBERS,
				ActionType:       models.REV_ACTION_DELETE,
				EntityAffectedId: employee.UserId,
				AdditionalInfo:   -1,
			}

			_, err = Store.AddEntityRevisionInTx(tnx, rev)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func _to_sh_branch_category(sp_branch_category *sp.BranchCategory) *models.ShBranchCategory {
	m_branch_category := new(models.ShBranchCategory)

	m_branch_category.BranchId = sp_branch_category.BranchId
	m_branch_category.CategoryId = sp_branch_category.CategoryId

	return m_branch_category
}

func applyBranchCategoryOperations(tnx *sql.Tx,
	posted_branch_categories []*sp.EntityRequest_RequestBranchCategory,
	old_2_new OLD_ID_2_NEW,
	company_id int64) error {

	for _, _p_branch_category := range posted_branch_categories {
		branch_category := _to_sh_branch_category(_p_branch_category.BranchCategory)
		branch_category.CompanyId = company_id

		branch_category.CategoryId = To_Server_Category_Id(branch_category.CategoryId)

		switch _p_branch_category.Action {
		case sp.EntityRequest_CREATE:
			branch_id, category_id := branch_category.BranchId, branch_category.CategoryId

			// if the id's of the branch category were the locally user generated(yet to be replaced with
			// server generated global ids), then use the server's
			if new_branch_id, ok := old_2_new.getType(_TYPE_BRANCH)[branch_id]; ok {
				branch_category.BranchId = new_branch_id
			}
			if new_category_id, ok := old_2_new.getType(_TYPE_CATEGORY)[category_id]; ok {
				branch_category.CategoryId = new_category_id
			}

			if _, err := Store.AddCategoryToBranchInTx(tnx, branch_category); err != nil {
				return fmt.Errorf("error adding category:%d to branch:%d '%s'",
					branch_category.CategoryId, branch_category.BranchId, err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_BRANCH_CATEGORY,
				ActionType:       models.REV_ACTION_CREATE,
				EntityAffectedId: branch_category.BranchId,
				AdditionalInfo:   branch_category.CategoryId,
			}

			if _, err := Store.AddEntityRevisionInTx(tnx, rev); err != nil {
				return err
			}

		case sp.EntityRequest_UPDATE:
		// TODO: not yet implemented b/c there is no "other-data" included in branch_category
		// there is not point in updating it

		case sp.EntityRequest_DELETE:
			branch_id, category_id := branch_category.BranchId, branch_category.CategoryId

			if _, err := Store.GetBranchCategoryInTx(tnx, branch_id, category_id); err == models.ErrNoData {
				continue
			} else if err != nil {
				return fmt.Errorf("error retriving branch category:(%d:%d) '%s'",
					branch_id, category_id, err.Error())
			}

			if err := Store.DeleteBranchCategoryInTx(tnx, branch_id, category_id); err != nil {
				return fmt.Errorf("error deleting branch_category: (%d:%d) '%s'",
					branch_id, category_id, err.Error())
			}

			rev := &models.ShEntityRevision{
				CompanyId:        company_id,
				EntityType:       models.REV_ENTITY_BRANCH_CATEGORY,
				ActionType:       models.REV_ACTION_DELETE,
				EntityAffectedId: branch_id,
				AdditionalInfo:   category_id,
			}

			if _, err := Store.AddEntityRevisionInTx(tnx, rev); err != nil {
				return err
			}
		}
	}

	return nil
}
