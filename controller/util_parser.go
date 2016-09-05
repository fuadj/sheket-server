package controller

import (
	"strings"
	"fmt"
	"strconv"
	"encoding/json"
)

func get_string(key string, check_map map[string]interface{}, does_exist map[string]bool) (string, bool) {
	if val, ok := check_map[key]; ok {
		s, ok := val.(string)
		if !ok {
			return "", false
		}
		if does_exist != nil {
			does_exist[key] = true
		}
		return s, true
	}
	return "", false
}

func get_bool(key string, check_map map[string]interface{}, does_exist map[string]bool) (bool, bool) {
	if val, ok := check_map[key]; ok {
		b, ok := val.(bool)
		if !ok {
			return false, false
		}
		if does_exist != nil {
			does_exist[key] = true
		}
		return b, true
	}
	return false, false
}

func get_int64(key string, check_map map[string]interface{}, does_exist map[string]bool) (int64, bool) {
	if val, ok := check_map[key]; ok {
		number, ok := val.(json.Number)
		if !ok {
			return -1, false
		}
		int_val, err := number.Int64()
		if err != nil {
			return -1, false
		}
		if does_exist != nil {
			does_exist[key] = true
		}
		return int_val, true
	}
	return -1, false
}

func get_float64(key string, check_map map[string]interface{}, does_exist map[string]bool) (float64, bool) {
	if val, ok := check_map[key]; ok {
		number, ok := val.(json.Number)
		if !ok {
			return -1, false
		}
		float_val, err := number.Float64()
		if err != nil {
			return -1, false
		}
		if does_exist != nil {
			does_exist[key] = true
		}
		return float_val, true
	}
	return -1, false
}

// Useful in map's as a key
// Without this, the key should be a 2-level thing
// e.g: map[outer_key]map[inner_key] object
type Pair_BranchItem struct {
	BranchId int64
	ItemId   int64
}

type Pair_BranchCategory struct {
	BranchId 	int64
	CategoryId 	int64
}

/**
 * split a string representing 2 integers split by a colon to its
 * component integer. Returns error if it can't do it.
 */
func splitColonSeparatedIntegers(s, err_msg string) (first, second int64, err error) {
	index := strings.Index(s, ":")
	if index == -1 {
		return first, second, fmt.Errorf("'%s' doesn't have : separator", s)
	}
	if index == 0 || index == (len(s)-1) {
		return first, second, fmt.Errorf("':' doesn't have one of the fields")
	}
	i, err := strconv.Atoi(s[:index])
	if err != nil {
		return first, second, err
	}
	first = int64(i)
	i, err = strconv.Atoi(s[index+1:])
	if err != nil {
		return first, second, err
	}
	second = int64(i)
	return first, second, nil
}

func toPair_BranchItem(s string) (result Pair_BranchItem, err error) {
	result.BranchId, result.ItemId, err = splitColonSeparatedIntegers(s,
		"id of branch_item doesn't split")
	return result, err
}

func toPair_BranchCategory(s string) (result Pair_BranchCategory, err error) {
	result.BranchId, result.CategoryId, err = splitColonSeparatedIntegers(s,
		"id of branch_category doesn't split")
	return result, err
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

func toPair_BranchCategorySet(arr []interface{}) (map[Pair_BranchCategory]bool, error) {
	set := make(map[Pair_BranchCategory]bool, len(arr))
	for i, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("branch_category:%d invalid id '%v'", i, v)
		}
		pair_branch_category, err := toPair_BranchCategory(s)
		if err != nil {
			return nil, err
		}
		set[pair_branch_category] = true
	}
	return set, nil
}

