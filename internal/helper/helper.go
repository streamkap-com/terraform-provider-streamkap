package helper

import "strconv"

func ParseJSONInt64(input any) int64 {
	var (
		fltInput float64
		strInput string
		output   int64
		ok       bool
	)

	strInput, ok = input.(string)

	if !ok {
		fltInput, _ = input.(float64)
		output = int64(fltInput)
	} else {
		output, _ = strconv.ParseInt(strInput, 10, 64)
	}

	return output
}
