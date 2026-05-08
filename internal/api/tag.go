package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

type GetTagResponse struct {
	Tags []Tag `json:"tags"`
}

type Tag struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        []string `json:"type"`
	System      bool     `json:"system,omitempty"`
	Custom      *bool    `json:"custom,omitempty"`
}

// TagListFilters scopes a ListTags call. Any field may be empty/nil to skip
// that filter; the backend AND-s populated filters together. Mirrors the query
// parameters on `GET /tags` and the body on `POST /tags/search`.
type TagListFilters struct {
	// Name filters by tag name (mapped to query param `tag_name`).
	Name string
	// Types filters by entity-type membership; matches tags whose `type` set
	// contains ANY of the given values (backend default tag_logic="or").
	Types []string
	// IDs filters by specific tag IDs. When the resulting GET URL would be
	// excessively long (many IDs), ListTags transparently switches to the
	// `POST /tags/search` body form.
	IDs []string
}

// tagsSearchBody mirrors backend `TagsSearchReqBody` for `POST /tags/search`.
type tagsSearchBody struct {
	TagIDs []string `json:"tag_ids,omitempty"`
}

// listTagsURLLengthThreshold — empirical safe URL length before we route
// through POST /tags/search. The backend documents the search endpoint as
// the alternative when "URL length limits are exceeded (e.g., 100+ tag IDs)";
// 1500 leaves headroom under common 8KB-ish proxies/CDNs.
const listTagsURLLengthThreshold = 1500

// ListTags returns tags for the current tenant filtered by name/types/ids.
// Routes through GET /tags by default; switches to POST /tags/search when the
// resulting GET URL would exceed listTagsURLLengthThreshold (large IDs lists).
func (s *streamkapAPI) ListTags(ctx context.Context, filters TagListFilters) ([]Tag, error) {
	parsedURL, err := url.Parse(s.cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	parsedURL = parsedURL.JoinPath("tags")

	q := parsedURL.Query()
	if filters.Name != "" {
		q.Set("tag_name", filters.Name)
	}
	for _, t := range filters.Types {
		q.Add("tag_type", t)
	}
	for _, id := range filters.IDs {
		q.Add("tag_ids", id)
	}
	parsedURL.RawQuery = q.Encode()

	useSearchBody := len(parsedURL.String()) > listTagsURLLengthThreshold

	var req *http.Request
	if useSearchBody {
		// Drop tag_ids from the URL — they go in the body instead. Other
		// filters (tag_name, tag_type) stay on the query string per backend
		// "body parameters take precedence over query parameters".
		q.Del("tag_ids")
		searchURL := *parsedURL
		searchURL.Path = parsedURL.Path + "/search"
		searchURL.RawQuery = q.Encode()

		body, err := json.Marshal(tagsSearchBody{TagIDs: filters.IDs})
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, searchURL.String(), bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		tflog.Debug(ctx, fmt.Sprintf(
			"ListTags (POST /tags/search) request details:\n"+
				"\tMethod: %s\n"+
				"\tURL: %s\n"+
				"\tBody: %s",
			req.Method,
			req.URL.String(),
			redactSensitiveJSON(body),
		))
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), http.NoBody)
		if err != nil {
			return nil, err
		}
		tflog.Debug(ctx, fmt.Sprintf(
			"ListTags request details:\n"+
				"\tMethod: %s\n"+
				"\tURL: %s\n",
			req.Method,
			req.URL.String(),
		))
	}

	var resp GetTagResponse
	if err := s.doRequest(ctx, req, &resp); err != nil {
		return nil, err
	}
	return resp.Tags, nil
}

func (s *streamkapAPI) GetTag(ctx context.Context, TagID string) (*Tag, error) {
	url, err := url.Parse(s.cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	url = url.JoinPath("tags")
	q := url.Query()
	q.Set("tag_ids", TagID)
	url.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetTagResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Tags) == 0 {
		return nil, nil
	}

	return &resp.Tags[0], nil
}

func (s *streamkapAPI) CreateTag(ctx context.Context, reqPayload Tag) (*Tag, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	var payloadMap map[string]any
	err = json.Unmarshal(payload, &payloadMap)
	if err != nil {
		return nil, err
	}

	payloadMap["created_from"] = constants.TERRAFORM

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/tags", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		redactSensitiveJSON(payload),
	))
	var resp Tag
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		// Mirror the adopt-on-exists pattern from sources/destinations/
		// transforms: when the backend reports the tag already exists, look
		// it up by name and return it instead of failing the apply. Without
		// this, a leaked `tf-acc-test-*` tag from a previous CI run blocks
		// every subsequent run until manually swept.
		//
		// Backend `create_custom_tag` raises `ValueError("A tag with
		// identical properties already exists.")` which the API layer wraps
		// into a 400 detail. Match either substring to be robust to phrasing
		// changes.
		if strings.Contains(err.Error(), "already exists") {
			tflog.Info(ctx, fmt.Sprintf(
				"Tag %q already exists — attempting to adopt existing resource", reqPayload.Name))
			adopted, adoptErr := s.adoptTagByName(ctx, reqPayload.Name)
			if adoptErr == nil {
				return adopted, nil
			}
			return nil, fmt.Errorf("%w (also tried to adopt the existing resource but could not locate it via the list endpoint: %v)", err, adoptErr)
		}
		return nil, err
	}
	return &resp, nil
}

// adoptTagByName looks up an existing tag by exact name. Used to recover
// from a 422-ish "already exists" response on Create when the upstream
// record was created out-of-band (or leaked by a previous test run).
func (s *streamkapAPI) adoptTagByName(ctx context.Context, name string) (*Tag, error) {
	tags, err := s.ListTags(ctx, TagListFilters{Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags while adopting %q: %w", name, err)
	}
	for i := range tags {
		if tags[i].Name == name {
			return &tags[i], nil
		}
	}
	return nil, fmt.Errorf("tag %q reported as existing but not found in list", name)
}

func (s *streamkapAPI) UpdateTag(ctx context.Context, tagID string, reqPayload Tag) (*Tag, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/tags/"+tagID, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		redactSensitiveJSON(payload),
	))
	var resp Tag
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *streamkapAPI) DeleteTag(ctx context.Context, tagID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/tags/"+tagID, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Tag
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}
	return nil
}
