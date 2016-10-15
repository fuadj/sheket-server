package controller

import (
	"golang.org/x/net/context"
	_ "net/http/httputil"
	"sheket/server/models"
	sp "sheket/server/sheketproto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
)

const (
	CLIENT_ROOT_CATEGORY_ID = -3
)

func To_Server_Category_Id(category_id int) int {
	if (category_id == CLIENT_ROOT_CATEGORY_ID) {
		return models.SERVER_ROOT_CATEGORY_ID
	} else {
		return category_id
	}
}

func To_Client_Category_Id(category_id int) int{
	if (category_id == models.SERVER_ROOT_CATEGORY_ID) {
		return CLIENT_ROOT_CATEGORY_ID
	} else {
		return category_id
	}
}

func (s *SheketController) SyncEntity(c context.Context, request *sp.EntityRequest) (response *sp.EntityResponse, err error) {
	defer trace("SyncEntity")()

	user_info, err := GetUserWithCompanyPermission(request.CompanyAuth)
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "%v", err)
	}

	tnx, err := Store.Begin()
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	var old_2_new OLD_ENTITY_ID_2_NEW

	if old_2_new, err = applyEntityOperations(tnx, request, user_info); err != nil {
		tnx.Rollback()
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}
	tnx.Commit()

	response = new(sp.EntityResponse)

	if err = fetchModifiedEntities(request, response, old_2_new, user_info); err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	for old_id, new_id := range old_2_new.getEntityType(_TYPE_ITEM) {
		updated := new(sp.EntityResponse_UpdatedId)
		updated.OldId = int32(old_id)
		updated.NewId = int32(new_id)
		response.UpdatedItemIds = append(response.UpdatedItemIds, updated)
	}
	for old_id, new_id := range old_2_new.getEntityType(_TYPE_BRANCH) {
		updated := new(sp.EntityResponse_UpdatedId)
		updated.OldId = int32(old_id)
		updated.NewId = int32(new_id)
		response.UpdatedBranchIds = append(response.UpdatedBranchIds, updated)
	}
	for old_id, new_id := range old_2_new.getEntityType(_TYPE_CATEGORY) {
		updated := new(sp.EntityResponse_UpdatedId)
		updated.OldId = int32(old_id)
		updated.NewId = int32(new_id)
		response.UpdatedCategoryIds = append(response.UpdatedCategoryIds, updated)
	}

	return response, nil
}

/**
 * Writes to the response any entities that have been (inserted/updated/deleted) since their
 * last respective revision. (e.g: it will sync any changes on branch_items that have occurred
 * since user's last branch_item revision).
 */
func fetchModifiedEntities(request *sp.EntityRequest,
	response *sp.EntityResponse,
	old_2_new OLD_ENTITY_ID_2_NEW,
	user_info *UserCompanyPermission) error {

	if err := fetchCategoriesSinceLastRev(request, response, user_info.CompanyId); err != nil {
		return err
	}

	if err := fetchItemsSinceLastRev(request, response, user_info.CompanyId); err != nil {
		return err
	}

	if err := fetchBranchesSinceLastRev(request, response, user_info.CompanyId); err != nil {
		return nil
	}

	if err := fetchBranchCategoriesSinceLastRev(request, response, user_info.CompanyId); err != nil {
		return err
	}

	if user_info.Permission.HasManagerAccess() {
		if err := fetchMembersSinceLastRev(request, response, user_info.CompanyId); err != nil {
			return err
		}
	}
	return nil
}

func fetchCategoriesSinceLastRev(request *sp.EntityRequest,
	response *sp.EntityResponse,
	company_id int) error {

	max_rev, category_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_CATEGORY,
			RevisionNumber: int(request.OldCategoryRev),
		})
	if err != nil && err != models.ErrNoData {
		return err
	}

	response.NewCategoryRev = int32(max_rev)

	for _, rev := range category_revs {
		category_id := rev.EntityAffectedId
		// TODO: check if it is newly created, don't re-fetch the category

		switch rev.ActionType {
		case models.REV_ACTION_CREATE, models.REV_ACTION_UPDATE:
			category, err := Store.GetCategoryById(category_id)
			if err != nil {
				if err == models.ErrNoData {
					continue
				} else {
					return err
				}
			}

			category.ParentId = To_Client_Category_Id(category.ParentId)

			response.Categories = append(response.Categories,
				&sp.EntityResponse_SyncCategory{
					Category: &sp.Category{
						CategoryId: int32(category.CategoryId),
						Name:       category.Name,
						ParentId:   int32(category.ParentId),
						UUID:       category.ClientUUID,

						// TODO: check if we support "hiding" categories
						StatusFlag: int32(models.STATUS_VISIBLE),
					},
				})

		case models.REV_ACTION_DELETE:
			response.Categories = append(response.Categories,
				&sp.EntityResponse_SyncCategory{
					Category: &sp.Category{
						CategoryId: int32(category_id),
					},
					State: sp.EntityResponse_REMOVED,
				})
		}
	}

	return nil
}

func fetchItemsSinceLastRev(request *sp.EntityRequest,
	response *sp.EntityResponse,
	company_id int) error {

	max_rev, changed_item_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_ITEM,
			RevisionNumber: int(request.OldItemRev),
		})
	if err != nil && err != models.ErrNoData {
		return err
	}

	response.NewItemRev = int32(max_rev)

	for _, item_rev := range changed_item_revs {
		item_id := item_rev.EntityAffectedId

		item, err := Store.GetItemById(item_id)
		if err != nil {
			if err == models.ErrNoData {
				continue
			} else {
				return err
			}
		}

		item.CategoryId = To_Client_Category_Id(item.CategoryId)

		response.Items = append(response.Items,
			&sp.EntityResponse_SyncItem{
				Item: &sp.Item{
					ItemId:            int32(item.ItemId),
					UUID:              item.ClientUUID,
					Name:              item.Name,
					Code:              item.ItemCode,
					CategoryId:        int32(item.CategoryId),
					UnitOfMeasurement: int32(item.UnitOfMeasurement),
					HasDerivedUnit:    item.HasDerivedUnit,
					DerivedName:       item.DerivedName,
					DerivedFactor:     item.DerivedFactor,
					StatusFlag:        int32(item.StatusFlag),
				},
			})
	}
	return nil
}

func fetchBranchesSinceLastRev(request *sp.EntityRequest,
	response *sp.EntityResponse, company_id int) error {

	max_rev, new_branch_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH,
			RevisionNumber: int(request.OldBranchRev),
		})
	if err != nil && err != models.ErrNoData {
		return err
	}

	response.NewBranchRev = int32(max_rev)

	for _, branch_rev := range new_branch_revs {
		branch_id := branch_rev.EntityAffectedId

		branch, err := Store.GetBranchById(branch_id)

		// it can be ErrNoData if the branch has been deleted since
		if err != nil {
			if err == models.ErrNoData {
				continue
			} else {
				return err
			}
		}

		response.Branches = append(response.Branches,
			&sp.EntityResponse_SyncBranch{
				Branch: &sp.Branch{
					BranchId:   int32(branch_id),
					UUID:       branch.ClientUUID,
					Name:       branch.Name,
					StatusFlag: int32(branch.StatusFlag),
				},
			})
	}
	return nil
}

func fetchMembersSinceLastRev(request *sp.EntityRequest,
	response *sp.EntityResponse,
	company_id int) error {

	max_rev, member_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_MEMBERS,
			RevisionNumber: int(request.OldMemberRev),
		})
	if err != nil && err != models.ErrNoData {
		return err
	}

	response.NewMemberRev = int32(max_rev)

	for _, rev := range member_revs {
		member_id := rev.EntityAffectedId

		switch rev.ActionType {
		case models.REV_ACTION_CREATE, models.REV_ACTION_UPDATE:
			user, err := Store.FindUserById(member_id)
			if err != nil {
				if err != models.ErrNoData {
					return err
				}
				continue
			}

			permission, err := Store.GetUserPermission(user, company_id)
			if err != nil {
				if err != models.ErrNoData {
					return err
				}
				continue
			}

			response.Employees = append(response.Employees,
				&sp.EntityResponse_SyncEmployee{
					Employee: &sp.Employee{
						EmployeeId: int32(member_id),
						Permission: permission.EncodedPermission,
						Name:       user.Username,
					},
				})

		case models.REV_ACTION_DELETE:
			response.Employees = append(response.Employees,
				&sp.EntityResponse_SyncEmployee{
					Employee: &sp.Employee{
						EmployeeId: int32(member_id),
					},
					State: sp.EntityResponse_REMOVED,
				})
		}
	}

	return nil
}

func fetchBranchCategoriesSinceLastRev(
	request *sp.EntityRequest,
	response *sp.EntityResponse,
	company_id int) error {

	max_rev, branch_category_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH_CATEGORY,
			RevisionNumber: int(request.OldBranchCategoryRev),
		})
	if err != nil && err != models.ErrNoData {
		return err
	}

	response.NewBranchCategoryRev = int32(max_rev)

	for _, rev := range branch_category_revs {
		branch_id := rev.EntityAffectedId
		category_id := rev.AdditionalInfo

		switch rev.ActionType {
		case models.REV_ACTION_CREATE, models.REV_ACTION_UPDATE:
			if _, err := Store.GetBranchCategory(branch_id, category_id); err != nil {
				if err == models.ErrNoData {
					continue
				}
				return err
			}

			response.BranchCategories = append(response.BranchCategories,
				&sp.EntityResponse_SyncBranchCategory{
					BranchCategory: &sp.BranchCategory{
						BranchId:   int32(branch_id),
						CategoryId: int32(To_Client_Category_Id(category_id)),
					},
				})
		case models.REV_ACTION_DELETE:
			response.BranchCategories = append(response.BranchCategories,
				&sp.EntityResponse_SyncBranchCategory{
					BranchCategory: &sp.BranchCategory{
						BranchId:   int32(branch_id),
						CategoryId: int32(To_Client_Category_Id(category_id)),
					},
					State: sp.EntityResponse_REMOVED,
				})
		}
	}

	return nil
}
