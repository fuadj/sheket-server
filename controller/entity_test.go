package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sheket/server/models"
	"strconv"
	"strings"
	"testing"
)

const (
	t_company_id      int64 = 10
	t_item_rev        int64 = 1231
	t_branch_rev      int64 = 12
	t_branch_item_rev int64 = 240
)

var syncJsonFormat string = `
	{
		"item_rev":%d,
		"branch_rev":%d,
		"branch_item_rev":%d,

		"types": [%s],

		%s
	}
`

var entityJsonFormat string = `
	"%s": {
		"create": [ %s ],
		"update": [ %s ],
		"delete": [ %s ],

		"fields": [ %s ]
	}
`

type intIds []int64

func (arr intIds) String() string {
	var result []string
	for _, v := range arr {
		result = append(result, strconv.Itoa(int(v)))
	}
	return strings.Join(result, ", ")
}

var itemsParseTest = []struct {
	entityType string
	create_ids intIds
	update_ids intIds
	delete_ids intIds

	existingItems []*models.ShItem
	fields        []map[string]interface{}
	wantResponse  int
}{
	{
		type_items,

		// create ids
		intIds{-1, -2, -7},
		// update ids
		intIds{77},
		// delete ids
		intIds{},
		[]*models.ShItem{
			&models.ShItem{ItemId: 77, CompanyId: t_company_id, ModelYear: "Old year"},
		},
		// fields
		[]map[string]interface{}{
			{
				models.ITEM_JSON_ITEM_ID:    -1,
				models.ITEM_JSON_COMPANY_ID: t_company_id,
				models.ITEM_JSON_MODEL_YEAR: "2007",
				models.ITEM_JSON_BAR_CODE:   "123456789",
			},
			{
				models.ITEM_JSON_ITEM_ID:     -2,
				models.ITEM_JSON_COMPANY_ID:  t_company_id,
				models.ITEM_JSON_MODEL_YEAR:  "1992",
				models.ITEM_JSON_MANUAL_CODE: "A-1028",
			},
			{
				models.ITEM_JSON_ITEM_ID:     77,
				models.ITEM_JSON_COMPANY_ID:  t_company_id,
				models.ITEM_JSON_MODEL_YEAR:  "new year",
				models.ITEM_JSON_MANUAL_CODE: "updated model",
			},
			{
				models.ITEM_JSON_ITEM_ID:     -7,
				models.ITEM_JSON_COMPANY_ID:  t_company_id,
				models.ITEM_JSON_PART_NUMBER: "52jk",
			},
		},
		http.StatusOK,
	},
	{
		type_items,
		intIds{-1, -7},
		intIds{},
		intIds{},
		[]*models.ShItem{},
		[]map[string]interface{}{
			{
				// doesn't exist in any of CRUD listings
				models.ITEM_JSON_ITEM_ID:     -4,
				models.ITEM_JSON_COMPANY_ID:  t_company_id,
				models.ITEM_JSON_MODEL_YEAR:  "1992",
				models.ITEM_JSON_MANUAL_CODE: "A-1028",
			},
			{
				models.ITEM_JSON_ITEM_ID:     -7,
				models.ITEM_JSON_COMPANY_ID:  t_company_id,
				models.ITEM_JSON_PART_NUMBER: "52jk",
			},
		},
		http.StatusInternalServerError,
	},
	{
		"jibberish",
		intIds{},
		intIds{},
		intIds{},
		[]*models.ShItem{},
		[]map[string]interface{}{},
		http.StatusBadRequest,
	},
}

func getItemJsonAtIndex(i int) string {
	entity_type := itemsParseTest[i].entityType
	create_ids := itemsParseTest[i].create_ids.String()
	update_ids := itemsParseTest[i].update_ids.String()
	delete_ids := itemsParseTest[i].delete_ids.String()

	fields := make([]string, len(itemsParseTest[i].fields))
	for j, field := range itemsParseTest[i].fields {
		s, err := json.MarshalIndent(field, "", "   ")
		if err != nil {
			return ""
		}
		fields[j] = string(s)
	}

	return fmt.Sprintf(entityJsonFormat, entity_type, create_ids,
		update_ids, delete_ids, strings.Join(fields, ", "))
}

func wrapType(t string) string { return fmt.Sprintf(`"%s"`, t) }

func TestEntityItemParser(t *testing.T) {
	for i, test := range itemsParseTest {
		s := fmt.Sprintf(syncJsonFormat,
			t_item_rev, t_branch_rev, t_branch_item_rev,
			wrapType(test.entityType),
			getItemJsonAtIndex(i))

		info := &IdentityInfo{CompanyId: t_company_id}
		_, err := parseEntityPost(strings.NewReader(s), parsers, info)
		if err != nil && test.wantResponse == http.StatusOK {
			t.Errorf("test %d failed with %s\n%v", i, s, err.Error())
			continue
		}
	}
}

func TestEntitySyncHandler(t *testing.T) {
	setup_store(t)
	defer teardown_store()

	setup_user(t, user_id)
	defer teardown_user()

	setup_tnx(t)
	defer teardown_tnx()

	source := models.NewMockSource(t_ctrl)
	t_mock.Source = source

	user_store := models.NewMockUserStore(t_ctrl)
	permission := &models.UserPermission{PermissionType: models.PERMISSION_TYPE_CREATOR}
	permission.Encode()
	t_mock.UserStore = user_store

	t_mock.BranchStore = models.NewSimpleBranchStore()
	t_mock.BranchItemStore = models.NewSimpleBranchItemStore(nil)
	t_mock.RevisionStore = models.NewSimpleRevisionStore(nil)

	for i, test := range itemsParseTest {
		// this is called for each test
		user_store.EXPECT().GetUserPermission(t_user, company_id).Return(permission, nil)
		if test.wantResponse != http.StatusBadRequest {
			// if there was a problem parsing, we won't get into the
			// creating the transaction stage.
			source.EXPECT().Begin().Return(t_tnx, nil)
		}

		t_mock.ItemStore = models.NewSimpleItemStore(test.existingItems)

		s := fmt.Sprintf(syncJsonFormat,
			t_item_rev, t_branch_rev, t_branch_item_rev,
			wrapType(test.entityType),
			getItemJsonAtIndex(i))
		req, err := http.NewRequest("POST", "www.example.com", strings.NewReader(s))
		if err != nil {
			t.Fatalf("request error '%v'", err)
		}
		req.Header.Set(JSON_KEY_COMPANY_ID, fmt.Sprintf("%d", company_id))
		w := httptest.NewRecorder()
		EntitySyncHandler(w, req)
		if w.Code != test.wantResponse {
			t.Errorf("Test:%d, Handler exited non expected code\n"+
				"wanted %s, got %s", i, http.StatusText(test.wantResponse),
				http.StatusText(w.Code))
			t.Errorf("Body :%s", w.Body.String())
		}
	}
}
