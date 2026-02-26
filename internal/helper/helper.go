package helper

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func GetTfCfgString(ctx context.Context, cfg map[string]any, key string) types.String {
	if val, ok := cfg[key]; ok && val != nil {
		strVal, ok := val.(string)
		if !ok {
			tflog.Warn(ctx, fmt.Sprintf("GetTfCfgString: expected string for key %q, got %T", key, val))
			return types.StringNull()
		}

		return types.StringValue(strVal)
	}

	return types.StringNull()
}

func GetTfCfgInt64(ctx context.Context, cfg map[string]any, key string) types.Int64 {
	if val, ok := cfg[key]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			intVal, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("GetTfCfgInt64: failed to parse string %q for key %q: %v", strVal, key, err))
				return types.Int64Null()
			}

			return types.Int64Value(intVal)
		}

		if floatVal, ok := val.(float64); ok {
			return types.Int64Value(int64(floatVal))
		}

		tflog.Warn(ctx, fmt.Sprintf("GetTfCfgInt64: expected string or float64 for key %q, got %T", key, val))
		return types.Int64Null()
	}

	return types.Int64Null()
}

func GetTfCfgBool(ctx context.Context, cfg map[string]any, key string) types.Bool {
	if val, ok := cfg[key]; ok && val != nil {
		boolVal, ok := val.(bool)
		if !ok {
			tflog.Warn(ctx, fmt.Sprintf("GetTfCfgBool: expected bool for key %q, got %T", key, val))
			return types.BoolNull()
		}

		return types.BoolValue(boolVal)
	}

	return types.BoolNull()
}

func GetTfCfgFloat64(ctx context.Context, cfg map[string]any, key string) types.Float64 {
	if val, ok := cfg[key]; ok && val != nil {
		if strVal, ok := val.(string); ok {
			floatVal, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("GetTfCfgFloat64: failed to parse string %q for key %q: %v", strVal, key, err))
				return types.Float64Null()
			}

			return types.Float64Value(floatVal)
		}

		if floatVal, ok := val.(float64); ok {
			return types.Float64Value(floatVal)
		}

		tflog.Warn(ctx, fmt.Sprintf("GetTfCfgFloat64: expected string or float64 for key %q, got %T", key, val))
		return types.Float64Null()
	}

	return types.Float64Null()
}

func GetTfCfgListString(ctx context.Context, cfg map[string]any, key string) types.List {
	if val, ok := cfg[key]; ok && val != nil {
		listVal, ok := val.([]interface{})
		if !ok {
			tflog.Warn(ctx, fmt.Sprintf("GetTfCfgListString: expected []interface{} for key %q, got %T", key, val))
			return types.ListNull(types.StringType)
		}

		result, diags := types.ListValueFrom(ctx, types.StringType, listVal)
		if diags.HasError() {
			tflog.Warn(ctx, fmt.Sprintf("GetTfCfgListString: failed to convert list for key %q: %s", key, diags.Errors()))
			return types.ListNull(types.StringType)
		}
		return result
	}

	return types.ListNull(types.StringType)
}

func GetTfCfgMapString(ctx context.Context, cfg map[string]any, key string) map[string]types.String {
	if val, ok := cfg[key]; ok && val != nil {
		// Handle map[string]interface{}
		if mapVal, ok := val.(map[string]interface{}); ok {
			result := make(map[string]types.String, len(mapVal))
			for k, v := range mapVal {
				if strVal, ok := v.(string); ok {
					result[k] = types.StringValue(strVal)
				} else {
					result[k] = types.StringNull()
				}
			}
			return result
		}
		// Handle map[string]string
		if mapVal, ok := val.(map[string]string); ok {
			result := make(map[string]types.String, len(mapVal))
			for k, v := range mapVal {
				result[k] = types.StringValue(v)
			}
			return result
		}

		tflog.Warn(ctx, fmt.Sprintf("GetTfCfgMapString: expected map type for key %q, got %T", key, val))
	}

	return nil
}
