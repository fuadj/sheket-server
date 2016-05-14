package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"io"
	"net/http"
	"net/http/httputil"
	"sheket/server/models"
	"strconv"
	"strings"
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

type CRUD_ACTION int64

const (
	ACTION_CREATE CRUD_ACTION = iota
	ACTION_UPDATE
	ACTION_DELETE
)

type EntitySyncData struct {
	RevisionItem        int64
	RevisionBranch      int64
	RevisionBranch_Item int64
	RevisionMember      int64

	// This holds the 'type' of items in the upload
	Types map[string]bool

	// Each CRUD operation has a "set" of ids it operates on
	// Those ids are then linked to objects affected
	ItemIds    map[CRUD_ACTION]map[int64]bool
	ItemFields map[int64]*SyncInventoryItem

	BranchIds    map[CRUD_ACTION]map[int64]bool
	BranchFields map[int64]*SyncBranch

	MemberIds    map[CRUD_ACTION]map[int64]bool
	MemberFields map[int64]*SyncMember

	Branch_ItemIds    map[CRUD_ACTION]map[Pair_BranchItem]bool
	Branch_ItemFields map[Pair_BranchItem]*SyncBranchItem
}

const (
	POST_TYPE_CREATE = iota
	POST_TYPE_UPDATE
	POST_TYPE_DELETE
)

type SuppliedFields struct {
	SetFields map[string]bool
}

type SyncInventoryItem struct {
	models.ShItem
	PostType int64

	SuppliedFields
}

type SyncBranch struct {
	models.ShBranch
	PostType int64

	SuppliedFields
}

type SyncBranchItem struct {
	models.ShBranchItem
	PostType int64

	SuppliedFields
}

type SyncMember struct {
	models.UserPermission
	PostType int64

	SuppliedFields
}

func CreateCRUDMaps(m map[CRUD_ACTION]map[int64]bool) {
	m[ACTION_CREATE] = make(map[int64]bool)
	m[ACTION_UPDATE] = make(map[int64]bool)
	m[ACTION_DELETE] = make(map[int64]bool)
}

func NewEntitySyncData() *EntitySyncData {
	s := &EntitySyncData{}

	s.Types = make(map[string]bool)
	s.ItemIds = make(map[CRUD_ACTION]map[int64]bool)
	s.ItemFields = make(map[int64]*SyncInventoryItem)

	s.BranchIds = make(map[CRUD_ACTION]map[int64]bool)
	s.BranchFields = make(map[int64]*SyncBranch)

	s.MemberIds = make(map[CRUD_ACTION]map[int64]bool)
	s.MemberFields = make(map[int64]*SyncMember)

	CreateCRUDMaps(s.ItemIds)
	CreateCRUDMaps(s.BranchIds)
	CreateCRUDMaps(s.MemberIds)

	s.Branch_ItemIds = make(map[CRUD_ACTION]map[Pair_BranchItem]bool)
	s.Branch_ItemIds[ACTION_CREATE] = make(map[Pair_BranchItem]bool)
	s.Branch_ItemIds[ACTION_UPDATE] = make(map[Pair_BranchItem]bool)
	s.Branch_ItemIds[ACTION_DELETE] = make(map[Pair_BranchItem]bool)
	s.Branch_ItemFields = make(map[Pair_BranchItem]*SyncBranchItem)

	return s
}

type IdentityInfo struct {
	CompanyId  int64
	User       *models.User
	Permission *models.UserPermission
}

// if it returns an error, parsing stops and error is propagated
type EntityParser func(*EntitySyncData, *simplejson.Json, *IdentityInfo) error

// parses an Entity post form the reader using the provided parsers for each entity type
func parseEntityPost(r io.Reader, parsers map[string]EntityParser, info *IdentityInfo) (*EntitySyncData, error) {
	data, err := simplejson.NewFromReader(r)
	if err != nil {
		return nil, err
	}

	entity_sync_data := NewEntitySyncData()
	entity_sync_data.RevisionItem = data.Get(key_item_revision).MustInt64(no_rev)
	entity_sync_data.RevisionBranch = data.Get(key_branch_revision).MustInt64(no_rev)
	// not used now, but might be needed in the future
	entity_sync_data.RevisionBranch_Item = data.Get(key_branch_item_rev).MustInt64(no_rev)
	entity_sync_data.RevisionMember = data.Get(key_member_revision).MustInt64(no_rev)

	for _, v := range data.Get(key_types).MustArray() {
		t, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type %v", v)
		}
		entity_sync_data.Types[t] = true
	}

	for e_type := range entity_sync_data.Types {
		body, ok := data.CheckGet(e_type)
		if !ok {
			return nil, fmt.Errorf("type %s doesn't have body", e_type)
		}
		if parser, ok := parsers[e_type]; ok {
			err := parser(entity_sync_data, body, info)

			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("type %s doesn't have parser installed", e_type)
		}
	}

	return entity_sync_data, nil
}

var parsers = map[string]EntityParser{
	type_items:        itemParser,
	type_branches:     branchParser,
	type_branch_items: branchItemParser,
	type_members:      memberParser,
}

// checks if the json has { create & update & delete } keys
func parserCRUDCheck(entity_name string, root *simplejson.Json) error {
	if _, ok := root.CheckGet(key_created); !ok {
		return fmt.Errorf("%s create key doesn't exist", entity_name)
	}
	if _, ok := root.CheckGet(key_updated); !ok {
		return fmt.Errorf("%s update key doesn't exist", entity_name)
	}
	if _, ok := root.CheckGet(key_deleted); !ok {
		return fmt.Errorf("%s delete key doesn't exist", entity_name)
	}

	return nil
}

// for entities with integer ids, grabs the ids of each CRUD operation
func parserCommon(entity_name string, root *simplejson.Json, action_ids map[CRUD_ACTION]map[int64]bool) error {
	if err := parserCRUDCheck(entity_name, root); err != nil {
		return err
	}

	int_arr, err := toIntArr(root.Get(key_created).MustArray())
	if err != nil {
		return err
	}
	action_ids[ACTION_CREATE] = intArrToSet(int_arr)
	int_arr, err = toIntArr(root.Get(key_updated).MustArray())
	if err != nil {
		return err
	}
	action_ids[ACTION_UPDATE] = intArrToSet(int_arr)
	int_arr, err = toIntArr(root.Get(key_deleted).MustArray())
	if err != nil {
		return err
	}
	action_ids[ACTION_DELETE] = intArrToSet(int_arr)

	if _, ok := root.CheckGet(key_fields); !ok {
		return fmt.Errorf("%s field doesn't exist", entity_name)
	}
	return nil
}

func itemParser(sync_data *EntitySyncData, root *simplejson.Json, info *IdentityInfo) error {
	if err := parserCommon("item", root, sync_data.ItemIds); err != nil {
		return err
	}

	// an array of items
	for _, v := range root.Get(key_fields).MustArray() {
		members, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid item fields %v", v)
		}

		item := &SyncInventoryItem{}
		item.SetFields = make(map[string]bool)

		var err error

		check_string_set := func(key string) (string, error) {
			if val, ok := members[key]; ok {
				s, ok := val.(string)
				if !ok {
					err = fmt.Errorf("invalid item.'%s' val %v", key, val)
					return "", err
				}
				item.SetFields[key] = true
				return s, nil
			}
			return "", nil
		}

		if val, ok := members[models.ITEM_JSON_ITEM_ID]; ok {
			item_id, ok := val.(json.Number)
			if !ok {
				return fmt.Errorf("invalid 'item_id' '%v'", val)
			}
			item.SetFields[models.ITEM_JSON_ITEM_ID] = true
			item.ItemId, err = item_id.Int64()
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("item_id missing %v", v)
		}

		item.CompanyId = info.CompanyId
		if item.Name, err = check_string_set(models.ITEM_JSON_ITEM_NAME); err != nil {
			return err
		}
		if item.ModelYear, err = check_string_set(models.ITEM_JSON_MODEL_YEAR); err != nil {
			return err
		}
		if item.PartNumber, err = check_string_set(models.ITEM_JSON_PART_NUMBER); err != nil {
			return err
		}
		if item.BarCode, err = check_string_set(models.ITEM_JSON_BAR_CODE); err != nil {
			return err
		}
		if item.ManualCode, err = check_string_set(models.ITEM_JSON_MANUAL_CODE); err != nil {
			return err
		}
		if item.ClientUUID, err = check_string_set(models.ITEM_JSON_UUID); err != nil {
			return err
		}

		if val, ok := members[models.ITEM_JSON_HAS_BAR_CODE]; ok {
			b, ok := val.(bool)
			if !ok {
				return fmt.Errorf("invalid 'has_bar_code' val %v", val)
			}
			item.SetFields[models.ITEM_JSON_HAS_BAR_CODE] = true
			item.HasBarCode = b
		}

		item_id := item.ItemId
		if sync_data.ItemIds[ACTION_CREATE][item_id] {
			item.PostType = POST_TYPE_CREATE
		} else if sync_data.ItemIds[ACTION_UPDATE][item_id] {
			item.PostType = POST_TYPE_UPDATE
		} else if sync_data.ItemIds[ACTION_DELETE][item_id] {
			item.PostType = POST_TYPE_DELETE
		} else {
			fmt.Errorf("item not listed in any of CRUD operations:%d", item_id)
		}

		sync_data.ItemFields[item_id] = item
	}

	return nil
}

func branchParser(sync_data *EntitySyncData, root *simplejson.Json, info *IdentityInfo) error {
	if err := parserCommon("branch", root, sync_data.BranchIds); err != nil {
		return err
	}

	for _, v := range root.Get(key_fields).MustArray() {
		members, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid branch fields %v", v)
		}

		branch := &SyncBranch{}
		branch.SetFields = make(map[string]bool)

		var err error

		check_string_set := func(key string) (string, error) {
			if val, ok := members[key]; ok {
				s, ok := val.(string)
				if !ok {
					return "", fmt.Errorf("invalid branch.'%s' val %v", key, val)
				}
				branch.SetFields[key] = true
				return s, nil
			}
			return "", nil
		}

		if val, ok := members[models.BRANCH_JSON_BRANCH_ID]; ok {
			branch_id, ok := val.(json.Number)
			if !ok {
				return fmt.Errorf("invalid 'branch_id' '%v'", val)
			}
			branch.SetFields[models.BRANCH_JSON_BRANCH_ID] = true
			branch.BranchId, err = branch_id.Int64()
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("branch_id missing %v", v)
		}

		branch.CompanyId = info.CompanyId

		if branch.Name, err = check_string_set(models.BRANCH_JSON_NAME); err != nil {
			return err
		}
		if branch.Location, err = check_string_set(models.BRANCH_JSON_LOCATION); err != nil {
			return err
		}
		if branch.ClientUUID, err = check_string_set(models.BRANCH_JSON_UUID); err != nil {
			return err
		}

		if sync_data.BranchIds[ACTION_CREATE][branch.BranchId] {
			branch.PostType = POST_TYPE_CREATE
		} else if sync_data.BranchIds[ACTION_UPDATE][branch.BranchId] {
			branch.PostType = POST_TYPE_UPDATE
		} else if sync_data.BranchIds[ACTION_DELETE][branch.BranchId] {
			branch.PostType = POST_TYPE_DELETE
		} else {
			fmt.Errorf("branch not listed in any of CRUD operations:%d", branch.BranchId)
		}

		sync_data.BranchFields[branch.BranchId] = branch
	}

	return nil
}

func toPair_BranchItem(s string) (Pair_BranchItem, error) {
	result := Pair_BranchItem{}
	index := strings.Index(s, ":")
	if index == -1 {
		return result, fmt.Errorf("'%s' doesn't have : separator", s)
	}
	if index == 0 || index == (len(s)-1) {
		return result, fmt.Errorf("branch_item id doesn't split around ':'")
	}
	branch_id, err := strconv.Atoi(s[:index])
	if err != nil {
		return result, err
	}
	item_id, err := strconv.Atoi(s[index+1:])
	if err != nil {
		return result, err
	}
	result.BranchId = int64(branch_id)
	result.ItemId = int64(item_id)
	return result, nil
}

func toPair_BranchItemSet(arr []interface{}) (map[Pair_BranchItem]bool, error) {
	set := make(map[Pair_BranchItem]bool, len(arr))
	for i, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("branch_item:%d invalid id '%v'", i, v)
		}
		pair_branch_item, err := toPair_BranchItem(s)
		if err != nil {
			return nil, err
		}
		set[pair_branch_item] = true
	}
	return set, nil
}

func branchItemParser(sync_data *EntitySyncData, root *simplejson.Json, info *IdentityInfo) error {
	var err error
	if err = parserCRUDCheck("branch_item", root); err != nil {
		return err
	}

	sync_data.Branch_ItemIds[ACTION_CREATE], err = toPair_BranchItemSet(
		root.Get(key_created).MustArray())
	if err != nil {
		return err
	}
	sync_data.Branch_ItemIds[ACTION_UPDATE], err = toPair_BranchItemSet(
		root.Get(key_updated).MustArray())
	if err != nil {
		return err
	}
	sync_data.Branch_ItemIds[ACTION_DELETE], err = toPair_BranchItemSet(
		root.Get(key_deleted).MustArray())
	if err != nil {
		return err
	}
	if _, ok := root.CheckGet(key_fields); !ok {
		return fmt.Errorf("branch_item_id fields doesn't exist")
	}

	for _, v := range root.Get(key_fields).MustArray() {
		members, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid branch item fields %v", v)
		}

		branch_item := &SyncBranchItem{}
		branch_item.SetFields = make(map[string]bool)

		var err error

		check_string_set := func(key string) (string, error) {
			if val, ok := members[key]; ok {
				s, ok := val.(string)
				if !ok {
					return "", fmt.Errorf("invalid branch_item.'%s' val %v", key, val)
				}
				branch_item.SetFields[key] = true
				return s, nil
			}
			return "", nil
		}

		var s_branch_item_id string
		if s_branch_item_id, err = check_string_set(models.BRANCH_ITEM_JSON_ID); err != nil {
			return err
		}
		pair_branch_item, err := toPair_BranchItem(s_branch_item_id)
		if err != nil {
			return err
		}
		branch_item.CompanyId = info.CompanyId
		branch_item.BranchId = pair_branch_item.BranchId
		branch_item.ItemId = pair_branch_item.ItemId

		if branch_item.ItemLocation, err = check_string_set(models.BRANCH_ITEM_JSON_ITEM_LOCATION); err != nil {
			return err
		}
		if val, ok := members[models.BRANCH_ITEM_JSON_QUANTITY]; ok {
			q, ok := val.(json.Number)
			if !ok {
				return fmt.Errorf("invalid 'quantity' val %v", val)
			}
			branch_item.SetFields[models.BRANCH_ITEM_JSON_QUANTITY] = true
			branch_item.Quantity, err = q.Float64()
			if err != nil {
				return err
			}
		}

		if sync_data.Branch_ItemIds[ACTION_CREATE][pair_branch_item] {
			branch_item.PostType = POST_TYPE_CREATE
		} else if sync_data.Branch_ItemIds[ACTION_UPDATE][pair_branch_item] {
			branch_item.PostType = POST_TYPE_UPDATE
		} else if sync_data.Branch_ItemIds[ACTION_DELETE][pair_branch_item] {
			branch_item.PostType = POST_TYPE_DELETE
		} else {
			fmt.Errorf("branch_item not listed in any of CRUD operations:%v", pair_branch_item)
		}

		sync_data.Branch_ItemFields[pair_branch_item] = branch_item
	}

	return nil
}

func memberParser(sync_data *EntitySyncData, root *simplejson.Json, info *IdentityInfo) error {
	if err := parserCommon("member", root, sync_data.MemberIds); err != nil {
		return err
	}

	for _, v := range root.Get(key_fields).MustArray() {
		fields, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Invalid member fields '%v'", v)
		}

		member := &SyncMember{}
		member.CompanyId = info.CompanyId
		member.SetFields = make(map[string]bool)

		var err error
		if val, ok := fields[models.PERMISSION_JSON_MEMBER_ID]; ok {
			member_id, ok := val.(json.Number)
			if !ok {
				return fmt.Errorf("invalid 'member_id' '%v'", val)
			}
			member.SetFields[models.PERMISSION_JSON_MEMBER_ID] = true
			if member.UserId, err = member_id.Int64(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("member_id missing %v", v)
		}

		if val, ok := fields[models.PERMISSION_JSON_MEMBER_PERMISSION]; ok {
			member.EncodedPermission, ok = val.(string)
			if !ok {
				return fmt.Errorf("invalid 'member_permission' '%v'", val)
			}
			member.SetFields[models.PERMISSION_JSON_MEMBER_PERMISSION] = true
		} else {
			return fmt.Errorf("member_permission missing %v", v)
		}

		if sync_data.MemberIds[ACTION_CREATE][member.UserId] {
			member.PostType = POST_TYPE_CREATE
		} else if sync_data.MemberIds[ACTION_UPDATE][member.UserId] {
			member.PostType = POST_TYPE_UPDATE
		} else if sync_data.MemberIds[ACTION_DELETE][member.UserId] {
			member.PostType = POST_TYPE_DELETE
		}

		sync_data.MemberFields[member.UserId] = member
	}

	return nil
}

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

	latest_item_rev, changed_items, err := fetchChangedItemsSinceRev(company_id,
		posted_data.RevisionItem, result.NewlyCreatedItemIds)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error()+"e2")
		return
	}
	sync_result[key_item_revision] = latest_item_rev
	if len(changed_items) > 0 {
		sync_result[key_sync_items] = changed_items
	}
	latest_branch_rev, changed_branches, err := fetchChangedBranchesSinceRev(company_id,
		posted_data.RevisionBranch, result.NewlyCreatedBranchIds)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error()+"e3")
		return
	}
	sync_result[key_branch_revision] = latest_branch_rev
	if len(changed_branches) > 0 {
		sync_result[key_sync_branches] = changed_branches
	}

	if permission.PermissionType <= models.PERMISSION_TYPE_BRANCH_MANAGER {
		max_member_rev, members, err := fetchChangedMemberSinceRev(company_id,
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
