package chainpot

import (
	"encoding/json"
	"strconv"
)

func ToString(v interface{}) string {
	if num, ok := v.(int); ok {
		return strconv.Itoa(num)
	} else if num, ok := v.(int64); ok {
		return strconv.Itoa(int(num))
	}
	return ""
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
