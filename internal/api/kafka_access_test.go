// internal/api/kafka_access_test.go
//
// Offline coverage for the Kafka-access and auth endpoints. These focus on the
// two places where the Go structs and the backend's Pydantic models disagree on
// the wire format, because each one silently corrupts state rather than failing
// loudly.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

// TestKafkaACL_TopicNameRoundTrips pins the asymmetry in the backend's ACL model:
// `topic_name` is only an input alias for a field named `name`, and responses are
// serialised by field name. Decoding `topic_name` alone left TopicName empty, which
// surfaces to users as "Provider produced inconsistent result after apply".
func TestKafkaACL_TopicNameRoundTrips(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"response uses the field name", `{"name":"orders","operation":"READ","resource_pattern_type":"LITERAL","resource":"TOPIC"}`},
		{"response uses the input alias", `{"topic_name":"orders","operation":"READ","resource_pattern_type":"LITERAL","resource":"TOPIC"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var acl KafkaACL
			require.NoError(t, json.Unmarshal([]byte(tt.body), &acl))
			assert.Equal(t, "orders", acl.TopicName)
			assert.Equal(t, "READ", acl.Operation)
			assert.Equal(t, "LITERAL", acl.ResourcePatternType)
			assert.Equal(t, "TOPIC", acl.Resource)
		})
	}
}

// TestKafkaACL_MarshalsWithAlias keeps requests on the documented public alias,
// which the backend accepts via populate_by_name.
func TestKafkaACL_MarshalsWithAlias(t *testing.T) {
	payload, err := json.Marshal(KafkaACL{
		TopicName:           "orders",
		Operation:           "READ",
		ResourcePatternType: "LITERAL",
		Resource:            "TOPIC",
	})
	require.NoError(t, err)

	var wire map[string]any
	require.NoError(t, json.Unmarshal(payload, &wire))
	assert.Equal(t, "orders", wire["topic_name"])
	assert.NotContains(t, wire, "name")
}

func TestCreateKafkaUser_SendsAliasAndDecodesFieldName(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	var captured map[string]any
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/kafka-access/kafka-users",
		func(req *http.Request) (*http.Response, error) {
			require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))
			// The backend echoes ACLs under the model field name, not the alias.
			return httpmock.NewJsonResponse(http.StatusOK, map[string]any{
				"username":             "tf-acc-kafkauser",
				"whitelist_ips":        "10.0.0.0/8",
				"kafka_proxy_endpoint": "proxy.example.com:9092",
				"kafka_acls": []map[string]any{
					{"name": "orders", "operation": "READ", "resource_pattern_type": "LITERAL", "resource": "TOPIC"},
				},
				"is_create_schema_registry": false,
			})
		},
	)

	user, err := client.CreateKafkaUser(context.Background(), CreateKafkaUserRequest{
		Username:     "tf-acc-kafkauser",
		Password:     "TestPassword123!",
		WhitelistIPs: "10.0.0.0/8",
		KafkaACLs: []KafkaACL{
			{TopicName: "orders", Operation: "READ", ResourcePatternType: "LITERAL", Resource: "TOPIC"},
		},
	})
	require.NoError(t, err)

	sentACLs, ok := captured["kafka_acls"].([]any)
	require.True(t, ok, "kafka_acls missing from request body")
	require.Len(t, sentACLs, 1)
	assert.Equal(t, "orders", sentACLs[0].(map[string]any)["topic_name"])
	assert.Equal(t, constants.TERRAFORM, captured["created_from"])

	require.Len(t, user.KafkaACLs, 1)
	assert.Equal(t, "orders", user.KafkaACLs[0].TopicName)
	assert.Equal(t, "proxy.example.com:9092", user.KafkaProxyEndpoint)
}

// TestGetKafkaUser_FiltersFromList covers the missing single-GET endpoint: Read
// has to scan the list, and a miss must be (nil, nil) so the resource is dropped
// from state instead of erroring.
func TestGetKafkaUser_FiltersFromList(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/kafka-access/kafka-users",
		httpmock.NewJsonResponderOrPanic(http.StatusOK, []map[string]any{
			{"username": "other-user", "kafka_proxy_endpoint": "a:9092", "kafka_acls": []any{}},
			{"username": "wanted-user", "kafka_proxy_endpoint": "b:9092", "kafka_acls": []any{}},
		}),
	)

	found, err := client.GetKafkaUser(context.Background(), "wanted-user")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "b:9092", found.KafkaProxyEndpoint)

	missing, err := client.GetKafkaUser(context.Background(), "no-such-user")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestUpdateClientCredential_PatchesRolesAndDescription(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	var captured map[string]any
	httpmock.RegisterResponder(
		http.MethodPatch,
		baseURL+"/auth/client-credentials/client-123",
		func(req *http.Request) (*http.Response, error) {
			require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))
			return httpmock.NewJsonResponse(http.StatusOK, map[string]any{
				"client_id":   "client-123",
				"description": "updated",
				"created_at":  "2026-08-25T10:00:00Z",
				// The backend persists and echoes a masked secret after creation.
				"secret": "abc******xyz",
				"roles": []map[string]any{
					{"id": "role-2", "key": "Admin", "name": "Admin", "description": "Full access"},
				},
			})
		},
	)

	cred, err := client.UpdateClientCredential(context.Background(), "client-123", UpdateClientCredentialRequest{
		RoleIDs:     []string{"role-2"},
		Description: "updated",
	})
	require.NoError(t, err)

	assert.Equal(t, []any{"role-2"}, captured["role_ids"])
	assert.Equal(t, "updated", captured["description"])
	// The backend rejects role_ids and permission_ids together.
	assert.NotContains(t, captured, "permission_ids")

	require.Len(t, cred.Roles, 1)
	assert.Equal(t, "role-2", cred.Roles[0].ID)
	assert.Equal(t, "updated", cred.Description)
}

func TestListRoles_Decodes(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/auth/roles",
		httpmock.NewJsonResponderOrPanic(http.StatusOK, []map[string]any{
			{"id": "role-1", "key": "ReadOnly", "name": "Read Only", "description": "Read access"},
			// description is nullable on the backend model.
			{"id": "role-2", "key": "Admin", "name": "Admin", "description": nil},
		}),
	)

	roles, err := client.ListRoles(context.Background())
	require.NoError(t, err)
	require.Len(t, roles, 2)
	assert.Equal(t, "ReadOnly", roles[0].Key)
	assert.Equal(t, "Read access", roles[0].Description)
	assert.Empty(t, roles[1].Description)
}
