package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (s *streamkapAPI) ListRoles(ctx context.Context) ([]Role, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/auth/roles", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"ListRoles request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp []Role
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
