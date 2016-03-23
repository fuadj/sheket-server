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

func _toInt64(i interface{}, def ...int64) int64 {
	if val, ok := i.(json.Number); ok {
		int_val, _ := val.Int64()
		return int_val
	}

	if len(def) > 0 {
		return def[0]
	}

	return int64(0)
}
