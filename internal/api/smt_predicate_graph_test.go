package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSMTPredicateGraphWireRoundTrip(t *testing.T) {
	cases := map[string]string{
		"A and B share new P": `{
  "catalog_precondition": {"contract_version": 0, "digest": ""},
  "expected_revision": "r1",
  "predicates": [{
    "request_key": "p",
    "type": "topic_name_matches",
    "config": {"pattern": "^orders\\..*$"},
    "secret_operations": [{"pointer": "/token", "operation": "replace", "value": "fixture-secret"}]
  }],
  "transforms": [
    {"id":"a","type":"regex_router","name":"A","enabled":true,"config":{"regex":"(.*)","replacement":"$1"},"predicate":{"request_key":"p","negate":false}},
    {"id":"b","type":"regex_router","name":"B","enabled":true,"config":{"regex":"(.*)","replacement":"$1"},"predicate":{"request_key":"p","negate":false}}
  ]
}`,
		"copy on write A to Q retains B on P without inheriting a secret": `{
  "catalog_precondition": {"contract_version": 0, "digest": ""},
  "expected_revision": "r2",
  "predicates": [
    {"id":"predicate-p"},
    {"request_key":"q","type":"topic_name_matches","config":{"pattern":"^payments\\..*$"}}
  ],
  "transforms": [
    {"id":"a","type":"regex_router","name":"A","enabled":true,"config":{"regex":"(.*)","replacement":"$1"},"predicate":{"request_key":"q","negate":false}},
    {"id":"b","type":"regex_router","name":"B","enabled":true,"config":{"regex":"(.*)","replacement":"$1"},"predicate":{"id":"predicate-p","negate":false}}
  ]
}`,
		"negate changes only the attachment": `{
  "catalog_precondition": {"contract_version": 0, "digest": ""},
  "expected_revision": "r3",
  "predicates": [{"id":"predicate-p"}],
  "transforms": [
    {"id":"b","type":"regex_router","name":"B","enabled":true,"config":{"regex":"(.*)","replacement":"$1"},"predicate":{"id":"predicate-p","negate":true}}
  ]
}`,
		"last reference removal garbage collects P": `{
  "catalog_precondition": {"contract_version": 0, "digest": ""},
  "expected_revision": "r4",
  "predicates": [],
  "transforms": []
}`,
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			raw := []byte(input)
			var write SMTChainWrite
			require.NoError(t, json.Unmarshal(raw, &write))
			encoded, err := json.Marshal(write)
			require.NoError(t, err)

			var want, got map[string]any
			require.NoError(t, json.Unmarshal(raw, &want))
			require.NoError(t, json.Unmarshal(encoded, &got))
			require.Equal(t, want, got, "predicate records and transform attachments must survive the provider wire DTO")
		})
	}
}

func TestSMTPredicateGraphReadRoundTrip(t *testing.T) {
	raw := []byte(`{
  "desired": {
    "chain_revision": "r2",
    "deployment_generation": 2,
    "predicates": [{
      "id": "predicate-p",
      "alias": "TopicNameMatches1",
      "type": "topic_name_matches",
      "schema_version": 1,
      "codec_version": 1,
      "config": {"pattern": "^orders\\..*$"},
      "configured_secret_paths": []
    }],
    "transforms": [{
      "kind": "typed",
      "id": "a",
      "alias": "RegexRouter1",
      "type": "regex_router",
      "schema_version": 1,
      "codec_version": 1,
      "name": "A",
      "enabled": true,
      "config": {"regex": "(.*)", "replacement": "$1"},
      "configured_secret_paths": [],
      "predicate": {"id": "predicate-p", "negate": false},
      "selector": null
    }],
    "managed": [],
    "managed_overrides": {},
    "order": [{"origin": "user", "reference": "a"}]
  },
  "last_applied": null,
  "application_status": "pending"
}`)

	var read SMTChainRead
	require.NoError(t, json.Unmarshal(raw, &read))
	encoded, err := json.Marshal(read)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(encoded, &got))
	desired := got["desired"].(map[string]any)
	require.Len(t, desired["predicates"], 1, "typed predicate records must not disappear on provider read")
}
