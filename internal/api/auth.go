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
	body := &GetAccessTokenRequest{
		ClientID: clientID,
		Secret:   secret,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/auth/access-token", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	var result Token
	err = s.doRequest(ctx, req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
