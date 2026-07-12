package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type Token struct {
	AccessToken  string `json:"accessToken"`
	Expires      string `json:"expires"`
	ExpiresIn    int64  `json:"expiresIn"`
	RefreshToken string `json:"refreshToken"`
}

type GetAccessTokenRequest struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"secret"`
}

func (s *streamkapAPI) GetAccessToken(clientID, secret string) (*Token, error) {
	ctx := context.Background()

	token, err := s.authenticate(ctx, clientID, secret)
	if err != nil {
		return nil, err
	}

	// Keep the credentials so the client can renew on its own. An apply that
	// runs longer than the token TTL (large estates, retry backoffs) otherwise
	// fails every remaining resource with an opaque 401.
	s.mu.Lock()
	s.clientID = clientID
	s.secret = secret
	s.mu.Unlock()

	return token, nil
}

// authenticate exchanges client credentials for an access token. It bypasses
// the 401-renewal path in do(): a 401 here means the credentials themselves are
// rejected, and renewing with the same credentials would recurse.
func (s *streamkapAPI) authenticate(ctx context.Context, clientID, secret string) (*Token, error) {
	payload, err := json.Marshal(&GetAccessTokenRequest{
		ClientID: clientID,
		Secret:   secret,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/auth/access-token", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	var result Token
	if err := s.do(ctx, req, &result, false); err != nil {
		return nil, err
	}

	return &result, nil
}
