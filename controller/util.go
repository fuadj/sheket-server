package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func trace(msg string) func() {
	start := time.Now()
	log.Printf("enter %s", msg)
	return func() { log.Printf("exit %s (%s)", msg, time.Since(start)) }
}

func writeErrorResponse(w http.ResponseWriter, err_code int, err_msg ...string) {
	w.WriteHeader(err_code)
	if len(err_msg) > 0 {
		w.Write([]byte(fmt.Sprintf(
			`{
			"error_message":%s,
			"error_code":%d
			}`, err_msg[0], err_code)))
	}
}

func toInt64(i interface{}, def ...int64) int64 {
	if val, ok := i.(json.Number); ok {
		int_val, _ := val.Int64()
		return int_val
	}

	if len(def) > 0 {
		return def[0]
	}

	return int64(0)
}

func toIntErr(i interface{}) (int64, error) {
	var err error
	if val, ok := i.(json.Number); ok {
		var int_val int64
		int_val, err = val.Int64()
		if err == nil {
			return int_val, nil
		}
	}
	return 0, fmt.Errorf("'%v' not an integer", i)
}

func toFloatErr(i interface{}) (float64, error) {
	var err error
	if val, ok := i.(json.Number); ok {
		var float_val float64
		float_val, err = val.Float64()
		if err == nil {
			return float_val, nil
		}
	}
	return 0, fmt.Errorf("'%v' not a float", i)
}

func toIntArr(iarr []interface{}) ([]int64, error) {
	result := make([]int64, len(iarr))
	for i, v := range iarr {
		int_val, err := toIntErr(v)
		if err != nil {
			return nil, err
		}
		result[i] = int_val
	}
	return result, nil
}

func intArrToSet(arr []int64) map[int64]bool {
	result := make(map[int64]bool, len(arr))
	for _, i := range arr {
		result[i] = true
	}
	return result
}
