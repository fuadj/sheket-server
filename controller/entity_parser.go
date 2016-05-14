package controller

import (
	"sheket/server/models"
	"github.com/bitly/go-simplejson"
	"io"
	"fmt"
	"encoding/json"
	"strings"
	"strconv"
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
	RevisionCategory    int64

	// This holds the 'type' of items in the upload
	Types map[string]bool

	// Each CRUD operation has a "set" of ids it operates on
	// Those ids are then linked to objects affected
	ItemIds    map[CRUD_ACTION]map[int64]bool
	ItemFields map[int64]*SyncInventoryItem

	CategoryIds    map[CRUD_ACTION]map[int64]bool
	CategoryFields map[int64]*SyncCategory

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

type SyncCategory struct {
	models.ShCategory
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

func NewEntitySyncData() *EntitySyncData {
	s := &EntitySyncData{}

	s.Types = make(map[string]bool)

	s.ItemIds = make(map[CRUD_ACTION]map[int64]bool)
	s.ItemFields = make(map[int64]*SyncInventoryItem)

	s.CategoryIds = make(map[CRUD_ACTION]map[int64]bool)
	s.CategoryFields = make(map[int64]*SyncInventoryItem)

	s.BranchIds = make(map[CRUD_ACTION]map[int64]bool)
	s.BranchFields = make(map[int64]*SyncBranch)

	s.MemberIds = make(map[CRUD_ACTION]map[int64]bool)
	s.MemberFields = make(map[int64]*SyncMember)

	initializeMap := func(m map[CRUD_ACTION]map[int64]bool) {
		m[ACTION_CREATE] = make(map[int64]bool)
		m[ACTION_UPDATE] = make(map[int64]bool)
		m[ACTION_DELETE] = make(map[int64]bool)
	}

	initializeMap(s.ItemIds)
	initializeMap(s.CategoryIds)
	initializeMap(s.BranchIds)
	initializeMap(s.MemberIds)

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
	entity_sync_data.RevisionMember = data.Get(key_member_revision).MustInt64(no_rev)
	entity_sync_data.RevisionCategory = data.Get(key_category_revision).MustInt64(no_rev)

	// not used now, but might be needed in the future
	entity_sync_data.RevisionBranch_Item = data.Get(key_branch_item_rev).MustInt64(no_rev)

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
	type_categories:   categoryParser,
}

// checks if the json has { create & update & delete } keys
func checkCRUDsExist(entity_name string, root *simplejson.Json) error {
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
func parseCRUDIntIds(entity_name string, root *simplejson.Json, entity_ids map[CRUD_ACTION]map[int64]bool) error {
	if err := checkCRUDsExist(entity_name, root); err != nil {
		return err
	}

	int_arr, err := toIntArr(root.Get(key_created).MustArray())
	if err != nil {
		return err
	}
	entity_ids[ACTION_CREATE] = intArrToSet(int_arr)
	int_arr, err = toIntArr(root.Get(key_updated).MustArray())
	if err != nil {
		return err
	}
	entity_ids[ACTION_UPDATE] = intArrToSet(int_arr)
	int_arr, err = toIntArr(root.Get(key_deleted).MustArray())
	if err != nil {
		return err
	}
	entity_ids[ACTION_DELETE] = intArrToSet(int_arr)

	if _, ok := root.CheckGet(key_fields); !ok {
		return fmt.Errorf("%s field doesn't exist", entity_name)
	}
	return nil
}

func itemParser(sync_data *EntitySyncData, root *simplejson.Json, info *IdentityInfo) error {
	if err := parseCRUDIntIds("item", root, sync_data.ItemIds); err != nil {
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
	if err := parseCRUDIntIds("branch", root, sync_data.BranchIds); err != nil {
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
	if err = checkCRUDsExist("branch_item", root); err != nil {
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
	if err := parseCRUDIntIds("member", root, sync_data.MemberIds); err != nil {
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

func categoryParser(sync_data *EntitySyncData, root *simplejson.Json, info *IdentityInfo) error {
	if err := parseCRUDIntIds("category", root, sync_data.CategoryIds); err != nil {
		return err
	}

	for _, v := range root.Get(key_fields).MustArray() {
		fields, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Invalid member fields '%v'", v)
		}

		category := &SyncCategory{}
		category.CompanyId = info.CompanyId
		category.SetFields = make(map[string]bool)

		var err error

		check_must_int_set := func(key string) (int64, error) {
			if val, ok := fields[key]; ok {
				v, ok := val.(json.Number)
				if !ok {
					return -1, fmt.Errorf("invalid category.%s '%v'", key, val)
				}
				category.SetFields[key] = true
				return v.Int64()
			}
			return -1, fmt.Errorf("category.%s not set", key)
		}

		check_string_set := func(key string) (string, error) {
			if val, ok := fields[key]; ok {
				s, ok := val.(string)
				if !ok {
					return "", fmt.Errorf("invalid category.'%s' val %v", key, val)
				}
				category.SetFields[key] = true
				return s, nil
			}
			return "", nil
		}

		if category.CategoryId, err = check_must_int_set(models.CATEGORY_JSON_CATEGORY_ID); err != nil {
			return fmt.Errorf("category.category_id missing '%s'", err.Error())
		}
		if category.ParentId, err = check_must_int_set(models.CATEGORY_JSON_PARENT_ID); err != nil {
			return fmt.Errorf("category.parent_id missing '%s'", err.Error())
		}

		if category.Name, err = check_string_set(models.CATEGORY_JSON_NAME); err != nil {
			return err
		}
		if category.ClientUUID, err = check_string_set(models.CATEGORY_JSON_UUID); err != nil {
			return err
		}

		if sync_data.CategoryIds[ACTION_CREATE][category.CategoryId] {
			category.PostType = POST_TYPE_CREATE
		} else if sync_data.CategoryIds[ACTION_UPDATE][category.CategoryId] {
			category.PostType = POST_TYPE_UPDATE
		} else if sync_data.CategoryIds[ACTION_DELETE][category.CategoryId] {
			category.PostType = ACTION_DELETE
		}

		sync_data.CategoryFields[category.CategoryId] = category
	}

	return nil
}
