package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
)

var ErrInvalidToken = errors.New("auth: invalid token")

type Validator interface {
	ValidatePlayerID(ctx context.Context, token string) (string, error)
}

type CognitoValidator struct {
	keySet jwk.Set
	issuer string
}

func NewCognitoValidator(ctx context.Context, jwksURL, issuer string) (*CognitoValidator, error) {
	keySet, err := fetchKeySet(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("auth: fetch jwks: %w", err)
	}
	return newCognitoValidator(keySet, issuer), nil
}

func newCognitoValidator(keySet jwk.Set, issuer string) *CognitoValidator {
	return &CognitoValidator{keySet: keySet, issuer: issuer}
}

func (v *CognitoValidator) ValidatePlayerID(ctx context.Context, token string) (string, error) {
	tok, err := jwt.Parse([]byte(token), jwt.WithKeySet(v.keySet), jwt.WithValidate(false))
	if err != nil {
		return "", ErrInvalidToken
	}
	if err := jwt.Validate(tok, jwt.WithIssuer(v.issuer)); err != nil {
		return "", ErrInvalidToken
	}

	sub, ok := tok.Subject()
	if !ok || sub == "" {
		return "", ErrInvalidToken
	}
	return sub, nil
}

func fetchKeySet(ctx context.Context, jwksURL string) (jwk.Set, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return jwk.Parse(body)
}
