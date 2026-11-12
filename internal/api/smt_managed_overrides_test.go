package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSMTChainWriteDistinguishesOverrideOmissionResetAndGraphOmission(t *testing.T) {
	cases := []struct {
		name           string
		write          SMTChainWrite
		wantOverrides  string
		wantTransforms bool
	}{
		{name: "unowned overrides", write: SMTChainWrite{Transforms: transformsPointer([]SMTInstanceWrite{})}, wantTransforms: true},
		{name: "reset overrides", write: SMTChainWrite{Transforms: transformsPointer([]SMTInstanceWrite{}), ManagedOverrides: mapPointer(map[string]bool{})}, wantOverrides: `{}`, wantTransforms: true},
		{name: "override only", write: SMTChainWrite{ManagedOverrides: mapPointer(map[string]bool{"user_defined:MaskField": false})}, wantOverrides: `{"user_defined:MaskField":false}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.write)
			require.NoError(t, err)
			var body map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &body))
			_, hasTransforms := body["transforms"]
			require.Equal(t, tc.wantTransforms, hasTransforms)
			if tc.wantOverrides == "" {
				require.NotContains(t, body, "managed_overrides")
			} else {
				require.JSONEq(t, tc.wantOverrides, string(body["managed_overrides"]))
			}
		})
	}
}

func mapPointer(value map[string]bool) *map[string]bool {
	return &value
}

func transformsPointer(value []SMTInstanceWrite) *[]SMTInstanceWrite {
	return &value
}
