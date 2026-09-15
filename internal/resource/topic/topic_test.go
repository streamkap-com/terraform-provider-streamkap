package topic

import (
	"errors"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"net/http"
	"testing"
)

func TestIsMissingTopic(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"missing", &api.APIError{StatusCode: http.StatusNotFound, Detail: "missing"}, true},
		{"legacy ownership", &api.APIError{StatusCode: http.StatusBadRequest, Detail: "Unauthorized. Topic is not owned by the tenant and/or service"}, true},
		{"legacy lookup", &api.APIError{StatusCode: http.StatusBadRequest, Detail: "Topic 'orders' not found in database"}, true},
		{"unrelated error", &api.APIError{StatusCode: http.StatusBadRequest, Detail: "broker not found"}, false},
		{"server error", &api.APIError{StatusCode: http.StatusInternalServerError, Detail: "Topic 'orders' not found in database"}, false},
		{"transport", errors.New("host not found"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMissingTopic(tc.err, "orders"); got != tc.want {
				t.Fatalf("isMissingTopic() = %t, want %t", got, tc.want)
			}
		})
	}
}
