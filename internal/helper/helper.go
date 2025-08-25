package helper

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func GetTfCfgString(cfg map[string]any, key string) types.String {
	if val, ok := cfg[key]; ok && val != nil {
		val, _ := val.(string)

		return types.StringValue(val)
	}

	return types.StringNull()
}

func GetTfCfgInt64(cfg map[string]any, key string) types.Int64 {
	if val, ok := cfg[key]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			val, _ := strconv.ParseInt(strVal, 10, 64)

			return types.Int64Value(val)
		} else {
			val, _ := val.(float64)

			return types.Int64Value(int64(val))
		}
	}

	return types.Int64Null()
}

func GetTfCfgBool(cfg map[string]any, key string) types.Bool {
	if val, ok := cfg[key]; ok && val != nil {
		val, _ := val.(bool)

		return types.BoolValue(val)
	}

	return types.BoolNull()
}

func GetTfCfgListString(ctx context.Context, cfg map[string]any, key string) types.List {
	if val, ok := cfg[key]; ok && val != nil {
		val, _ := val.([]interface{})

		listVal, _ := types.ListValueFrom(ctx, types.StringType, val)
		return listVal
	}

	return types.ListNull(types.StringType)
}
