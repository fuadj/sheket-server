package controller

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"io"
	"sheket/server/models"
	"strconv"
	"strings"
)

const (
	key_item_revision   = "item_rev"
	key_branch_revision = "branch_rev"

	key_types = "types"

	key_created = "create"
	key_updated = "update"
	key_deleted = "delete"

	key_fields = "fields"

	type_items        = "items"
	type_branches     = "branches"
	type_branch_items = "branch_items"
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

	// This holds the 'type' of items in the upload
	Types map[string]bool

	// Each CRUD operation has a "set" of ids it operates on
	// Those ids are then linked to objects affected
	ItemIds    map[CRUD_ACTION]map[int64]bool
	ItemFields map[int64]*SyncInventoryItem

	BranchIds    map[CRUD_ACTION]map[int64]bool
	BranchFields map[int64]*SyncBranch

	Branch_ItemIds    map[CRUD_ACTION]map[Pair_BranchItem]bool
	Branch_ItemFields map[Pair_BranchItem]*CachedBranchItem
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

	// This is especially useful in update mode
	SuppliedFields
}

type SyncBranch struct {
	models.ShBranch
	PostType int64

	// This is especially useful in update mode
	SuppliedFields
}

type SyncBranchItem struct {
	models.ShBranchItem
	PostType int64

	// This is especially useful in update mode
	SuppliedFields
}

func CreateSubMaps(m map[CRUD_ACTION]map[int64]bool, size int) {
	m[ACTION_CREATE] = make(map[int64]bool, size)
	m[ACTION_UPDATE] = make(map[int64]bool, size)
	m[ACTION_DELETE] = make(map[int64]bool, size)
}

func NewEntitySyncData() *EntitySyncData {
	s := &EntitySyncData{}
	s.Types = make(map[string]bool, 3)
	DEFAULT_SIZE := 10
	s.ItemIds = make(map[CRUD_ACTION]map[int64]bool, DEFAULT_SIZE)
	s.ItemFields = make(map[int64]*SyncInventoryItem, DEFAULT_SIZE)

	s.BranchIds = make(map[CRUD_ACTION]map[int64]bool, DEFAULT_SIZE)
	s.BranchFields = make(map[int64]*SyncBranch, DEFAULT_SIZE)

	CreateSubMaps(s.ItemIds, DEFAULT_SIZE)
	CreateSubMaps(s.BranchIds, DEFAULT_SIZE)

	s.Branch_ItemIds = make(map[CRUD_ACTION]map[Pair_BranchItem]bool, DEFAULT_SIZE)
	s.Branch_ItemIds[ACTION_CREATE] = make(map[Pair_BranchItem]bool, DEFAULT_SIZE)
	s.Branch_ItemIds[ACTION_UPDATE] = make(map[Pair_BranchItem]bool, DEFAULT_SIZE)
	s.Branch_ItemIds[ACTION_DELETE] = make(map[Pair_BranchItem]bool, DEFAULT_SIZE)
	s.Branch_ItemFields = make(map[Pair_BranchItem]*CachedBranchItem, DEFAULT_SIZE)

	return s
}

type IdentityInfo struct {
	CompanyId int64
	UserId    int64
}

// if it returns an error, parsing stops and error is propagated
type EntityParser func(*EntitySyncData, *simplejson.Json, *IdentityInfo) error

// parses an Entity post form the reader using the provided parsers for each entity type
func parseEntityPost(r io.Reader, parsers map[string]EntityParser, info *IdentityInfo) (*EntitySyncData, error) {
	data, err := simplejson.NewFromReader(r)
	if err != nil {
		return nil, err
	}

	entity_data := NewEntitySyncData()
	entity_data.RevisionItem = data.Get(key_item_revision).MustInt64(no_rev)
	entity_data.RevisionBranch = data.Get(key_branch_revision).MustInt64(no_rev)
	// not used now, but might be needed in the future
	entity_data.RevisionBranch_Item = data.Get(key_branch_item_rev).MustInt64(no_rev)

	for _, v := range data.Get(key_types).MustArray() {
		t, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type %v", v)
		}
		entity_data.Types[t] = true
	}

	for e_type := range entity_data.Types {
		body, ok := data.CheckGet(e_type)
		if !ok {
			return nil, fmt.Errorf("type %s doesn't have body", e_type)
		}
		err := parsers[e_type](entity_data, body, info)

		if err != nil {
			return nil, err
		}
	}

	return entity_data, nil
}

var parsers = map[string]EntityParser{
	type_items:        itemParser,
	type_branches:     branchParser,
	type_branch_items: branchItemParser,
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

	for k, v := range root.Get(key_fields).MustMap() {
		s, ok := k.(string)
		if !ok {
			return fmt.Errorf("invalid item id %v", k)
		}
		item_id, err := strconv.ParseInt(s, 10, 64)
		if err != err {
			return err
		}

		members, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid item fields %v", v)
		}

		item := &SyncInventoryItem{}
		item.SetFields = make(map[string]bool)
		if sync_data.ItemIds[ACTION_CREATE][item_id] {
			item.PostType = POST_TYPE_CREATE
		} else if sync_data.ItemIds[ACTION_UPDATE][item_id] {
			item.PostType = POST_TYPE_UPDATE
		} else if sync_data.ItemIds[ACTION_DELETE][item_id] {
			item.PostType = POST_TYPE_DELETE
		} else {
			fmt.Errorf("item not listed in any of CRUD operations:%d", item_id)
		}

		check_string_set := func(key string) string {
			if val, ok := members[key]; ok {
				s, ok := val.(string)
				if !ok {
					err = fmt.Errorf("invalid '%s' val %v", key, val)
					return
				}
				item.SetFields[key] = true
				return s
			}
			return ""
		}

		item.CompanyId = info.CompanyId
		item.ItemId = item_id
		item.ModelYear = check_string_set("model_year")
		if err != nil {
			return err
		}
		item.PartNumber = check_string_set("part_number")
		if err != nil {
			return err
		}
		item.BarCode = check_string_set("bar_code")
		if err != nil {
			return err
		}
		item.ManualCode = check_string_set("manual_code")
		if err != nil {
			return err
		}

		if val, ok := members["has_bar_code"]; ok {
			b, ok := val.(bool)
			if !ok {
				return fmt.Errorf("invalid 'has_bar_code' val %v", val)
			}
			item.SetFields["has_bar_code"] = true
			item.HasBarCode = b
		}

		sync_data.ItemFields[item_id] = item
	}

	return nil
}

func branchParser(sync_data *EntitySyncData, root *simplejson.Json, info *IdentityInfo) error {
	if err := parserCommon("branch", root, sync_data.BranchIds); err != nil {
		return err
	}

	for k, v := range root.Get(key_fields).MustMap() {
		s, ok := k.(string)
		if !ok {
			return fmt.Errorf("invalid branch id %v", k)
		}
		branch_id, err := strconv.ParseInt(s, 10, 64)
		if err != err {
			return err
		}

		members, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid branch fields %v", v)
		}

		branch := &SyncBranch{}
		branch.SetFields = make(map[string]bool)
		if sync_data.BranchIds[ACTION_CREATE][branch_id] {
			branch.PostType = POST_TYPE_CREATE
		} else if sync_data.BranchIds[ACTION_UPDATE][branch_id] {
			branch.PostType = POST_TYPE_UPDATE
		} else if sync_data.BranchIds[ACTION_DELETE][branch_id] {
			branch.PostType = POST_TYPE_DELETE
		} else {
			fmt.Errorf("branch not listed in any of CRUD operations:%d", branch_id)
		}

		check_string_set := func(key string) string {
			if val, ok := members[key]; ok {
				s, ok := val.(string)
				if !ok {
					err = fmt.Errorf("invalid '%s' val %v", key, val)
					return
				}
				branch.SetFields[key] = true
				return s
			}
			return ""
		}

		branch.CompanyId = info.CompanyId
		branch.BranchId = branch_id

		branch.Name = check_string_set("name")
		if err != nil {
			return err
		}
		branch.Location = check_string_set("location")
		if err != nil {
			return err
		}

		sync_data.BranchFields[branch_id] = branch
	}

	return nil
}

func toPair_BranchItem(s string) (Pair_BranchItem, error) {
	index := strings.Index(s, ":")
	if index == -1 {
		return nil, fmt.Errorf("'%s' doesn't have : separator", s)
	}
	if index == 0 || index == (len(s) - 1) {
		return nil, fmt.Errorf("branch_item id doesn't split around ':'")
	}
	branch_id, err := strconv.Atoi(s[:index])
	if err != nil {
		return nil, err
	}
	item_id, err := strconv.Atoi(s[index+1:])
	if err != nil {
		return nil, err
	}
	return Pair_BranchItem{int64(branch_id), int64(item_id)}, nil
}

func toPair_BranchItemSet(arr []interface{}) (set map[Pair_BranchItem]bool, error) {
	set = make(map[Pair_BranchItem]bool, len(arr))
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

	for k, v := range root.Get(key_fields).MustMap() {
		s, ok := k.(string)
		if !ok {
			return fmt.Errorf("invalid branch id %v", k)
		}
		pair_branch_item, err := toPair_BranchItem(s)
		if err != nil {
			return err
		}

		members, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid branch fields %v", v)
		}

		branch_item := &SyncBranchItem{}
		branch_item.SetFields = make(map[string]bool)
		if sync_data.BranchIds[ACTION_CREATE][pair_branch_item] {
			branch_item.PostType = POST_TYPE_CREATE
		} else if sync_data.BranchIds[ACTION_UPDATE][pair_branch_item] {
			branch_item.PostType = POST_TYPE_UPDATE
		} else if sync_data.BranchIds[ACTION_DELETE][pair_branch_item] {
			branch_item.PostType = POST_TYPE_DELETE
		} else {
			fmt.Errorf("branch_item not listed in any of CRUD operations:%v", pair_branch_item)
		}

		branch_item.CompanyId = info.CompanyId
		branch_item.BranchId = pair_branch_item.BranchId
		branch_item.ItemId = pair_branch_item.ItemId

		if val, ok := members["item_location"]; ok {
			loc, ok := val.(string)
			if !ok {
				return fmt.Errorf("invalid 'item_location' val %v", loc)
			}
			branch_item.SetFields["item_location"] = true
			branch_item.ItemLocation = loc
		}
		if val, ok := members["quantity"]; ok {
			q, ok := val.(float64)
			if !ok {
				return fmt.Errorf("invalid 'quantity' val %v", val)
			}
			branch_item.SetFields["quantity"] = true
			branch_item.Quantity = q
		}

		sync_data.Branch_ItemFields[pair_branch_item] = branch_item
	}

	return nil
}

