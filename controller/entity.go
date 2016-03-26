package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"io"
	"net/http"
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

	// used in the response json to hold the newly updated item ids
	key_updated_item_ids = "updated_item_ids"

	// used in the response json to hold the newly updated branch ids
	key_updated_branch_ids = "updated_branch_ids"

	// key of json holding any updated items since last sync
	key_sync_items = "sync_items"

	// key of json holding any updated branches since last sync
	key_sync_branches = "sync_branches"
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
	s.Branch_ItemFields = make(map[Pair_BranchItem]*SyncBranchItem, DEFAULT_SIZE)

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

		var s_branch_id string
		if s_branch_id, err = check_string_set(models.BRANCH_JSON_BRANCH_ID); err != nil {
			return err
		}

		branch_id, err := strconv.ParseInt(s_branch_id, 10, 64)
		if err != err {
			return err
		}

		branch.BranchId = branch_id
		branch.CompanyId = info.CompanyId
		branch.BranchId = branch_id

		if branch.Name, err = check_string_set(models.BRANCH_JSON_NAME); err != nil {
			return err
		}
		if branch.Location, err = check_string_set(models.BRANCH_JSON_LOCATION); err != nil {
			return err
		}

		if sync_data.BranchIds[ACTION_CREATE][branch_id] {
			branch.PostType = POST_TYPE_CREATE
		} else if sync_data.BranchIds[ACTION_UPDATE][branch_id] {
			branch.PostType = POST_TYPE_UPDATE
		} else if sync_data.BranchIds[ACTION_DELETE][branch_id] {
			branch.PostType = POST_TYPE_DELETE
		} else {
			fmt.Errorf("branch not listed in any of CRUD operations:%d", branch_id)
		}

		sync_data.BranchFields[branch_id] = branch
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
			return fmt.Errorf("invalid branch fields %v", v)
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
			q, ok := val.(float64)
			if !ok {
				return fmt.Errorf("invalid 'quantity' val %v", val)
			}
			branch_item.SetFields[models.BRANCH_ITEM_JSON_QUANTITY] = true
			branch_item.Quantity = q
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

type EntityResult struct {
	OldId2New_Items    map[int64]int64
	OldId2New_Branches map[int64]int64
}

func EntitySyncHandler(w http.ResponseWriter, r *http.Request) {
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

	sync_result := make(map[string]interface{})

	tnx, err := Store.Begin()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}

	result, err := applyEntityOperations(tnx, posted_data, info)
	if err != nil {
		tnx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	tnx.Commit()

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
		posted_data.RevisionItem)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}
	sync_result[key_item_revision] = latest_item_rev
	if len(changed_items) > 0 {
		sync_result[key_sync_items] = changed_items
	}
	latest_branch_rev, changed_branches, err := fetchChangedBranchesSinceRev(company_id,
		posted_data.RevisionBranch)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}
	sync_result[key_branch_revision] = latest_branch_rev
	if len(changed_branches) > 0 {
		sync_result[key_sync_branches] = changed_branches
	}

	b, err := json.MarshalIndent(sync_result, "", "    ")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func applyEntityOperations(tnx *sql.Tx, posted_data *EntitySyncData, info *IdentityInfo) (*EntityResult, error) {
	result := &EntityResult{}
	result.OldId2New_Items = make(map[int64]int64, 10)
	result.OldId2New_Branches = make(map[int64]int64, 10)

	// there were items in the post
	if len(posted_data.ItemFields) > 0 {
		// for the newly created items, get a globally after adding them to datastore
		for old_item_id := range posted_data.ItemIds[ACTION_CREATE] {
			item, ok := posted_data.ItemFields[old_item_id]
			if !ok {
				return nil, fmt.Errorf("item:%d doesn't have members defined", old_item_id)
			}
			created_item, err := Store.CreateItemInTx(tnx, &item.ShItem)
			if err != nil {
				return nil, fmt.Errorf("error creating item %s", err.Error())
			}
			result.OldId2New_Items[old_item_id] = created_item.ItemId
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
				return nil, fmt.Errorf("error retriving item:%d", item_id)
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
		}
		// TODO: delete item not yet implemented
	}

	// if there were branches in the post
	if len(posted_data.BranchFields) > 0 {
		for old_branch_id := range posted_data.BranchIds[ACTION_CREATE] {
			branch, ok := posted_data.BranchFields[old_branch_id]
			if !ok {
				return nil, fmt.Errorf("branch:%d doesn't have members defined")
			}
			created_branch, err := Store.CreateBranchInTx(tnx, &branch.ShBranch)
			if err != nil {
				return nil, fmt.Errorf("error creating branch %s", err.Error())
			}
			result.OldId2New_Branches[old_branch_id] = created_branch.BranchId
		}
		for branch_id := range posted_data.BranchIds[ACTION_UPDATE] {
			branch, ok := posted_data.BranchFields[branch_id]
			if !ok {
				return nil, fmt.Errorf("branch:%d doesn't have members defined")
			}
			previous_branch, err := Store.GetBranchByIdInTx(tnx, branch_id)
			if err != nil {
				return nil, fmt.Errorf("error retriving branch:%d", branch_id)
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
		}
		// TODO: delete branch not yet implemented
	}

	// if there were branch items in the post
	if len(posted_data.Branch_ItemFields) > 0 {
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

			if _, err := Store.AddItemToBranchInTx(tnx, &branch_item.ShBranchItem); err != nil {
				return nil, fmt.Errorf("error adding item:%d to branch:%d '%s'",
					branch_item.ItemId, branch_item.BranchId, err.Error())
			}
		}
		for pair_branch_item := range posted_data.Branch_ItemIds[ACTION_UPDATE] {
			posted_branch_item, ok := posted_data.Branch_ItemFields[pair_branch_item]
			if !ok {
				return nil, fmt.Errorf("branchItem:(%d,%d) doesn't have members defined",
					pair_branch_item.BranchId, pair_branch_item.ItemId)
			}
			branch_id, item_id := pair_branch_item.BranchId, pair_branch_item.ItemId

			previous_item, err := Store.GetBranchItemInTx(tnx, branch_id, item_id)
			if err != nil {
				return nil, fmt.Errorf("error retriving branchItem:(%d,%d)", branch_id, item_id)
			}

			if posted_branch_item.SetFields[models.BRANCH_ITEM_JSON_QUANTITY] {
				previous_item.Quantity = posted_branch_item.Quantity
			}
			if posted_branch_item.SetFields[models.BRANCH_ITEM_JSON_ITEM_LOCATION] {
				previous_item.ItemLocation = posted_branch_item.ItemLocation
			}

			if _, err = Store.UpdateBranchItemInTx(tnx, previous_item); err != nil {
				return nil, fmt.Errorf("error updating branchItem:(%d,%d) '%v'",
					branch_id, item_id, err.Error())
			}
		}
		// TODO: delete branch item not yet implemented
	}
	return result, nil
}

func fetchChangedItemsSinceRev(company_id, item_rev int64) (latest_rev int64,
	items_since []map[string]interface{}, err error) {

	max_rev, new_item_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_ITEM,
			RevisionNumber: item_rev,
		})
	if err != nil {
		return max_rev, nil, err
	}

	result := make([]map[string]interface{}, len(new_item_revs))
	i := 0
	for _, item_rev := range new_item_revs {
		item_id := item_rev.EntityAffectedId

		item, err := Store.GetItemById(item_id)
		if err != nil {
			continue
		}

		result[i] = map[string]interface{}{
			models.ITEM_JSON_ITEM_ID:      item.ItemId,
			models.ITEM_JSON_COMPANY_ID:   item.CompanyId,
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

func fetchChangedBranchesSinceRev(company_id, branch_rev int64) (latest_rev int64,
	branches_since []map[string]interface{}, err error) {

	max_rev, new_item_revs, err := Store.GetRevisionsSince(
		&models.ShEntityRevision{
			CompanyId:      company_id,
			EntityType:     models.REV_ENTITY_BRANCH,
			RevisionNumber: branch_rev,
		})
	if err != nil {
		return max_rev, nil, err
	}
	result := make([]map[string]interface{}, len(new_item_revs))

	i := 0
	for _, item_rev := range new_item_revs {
		branch_id := item_rev.EntityAffectedId

		branch, err := Store.GetBranchById(branch_id)
		if err != nil {
			// if a branch has been deleted in future revisions, we won't see it
			// so skip any error valued branches, and only return to the user those
			// that are correctly fetched
			continue
		}

		result[i] = map[string]interface{}{
			models.BRANCH_JSON_COMPANY_ID: branch.CompanyId,
			models.BRANCH_JSON_BRANCH_ID:  branch.BranchId,
			models.BRANCH_JSON_NAME:       branch.Name,
			models.BRANCH_JSON_LOCATION:   branch.Location,
		}
		i++
	}
	return max_rev, result[:i], nil
}
