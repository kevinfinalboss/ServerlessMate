package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testIssuer = "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_test"

type testKeyPair struct {
	private jwk.Key
	public  jwk.Key
}

func generateTestKeyPair(t *testing.T, kid string) testKeyPair {
	t.Helper()

	rawPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	priv, err := jwk.Import[jwk.Key](rawPriv)
	require.NoError(t, err)
	require.NoError(t, priv.Set(jwk.KeyIDKey, kid))
	require.NoError(t, priv.Set(jwk.AlgorithmKey, jwa.RS256()))

	pub, err := jwk.PublicKeyOf(priv)
	require.NoError(t, err)
	require.NoError(t, pub.Set(jwk.KeyIDKey, kid))
	require.NoError(t, pub.Set(jwk.AlgorithmKey, jwa.RS256()))

	return testKeyPair{private: priv, public: pub}
}

func signToken(t *testing.T, key jwk.Key, subject, issuer string, expiresAt time.Time) string {
	t.Helper()

	tok, err := jwt.NewBuilder().
		Subject(subject).
		Issuer(issuer).
		IssuedAt(time.Now().Add(-time.Minute)).
		Expiration(expiresAt).
		Build()
	require.NoError(t, err)

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), key))
	require.NoError(t, err)
	return string(signed)
}

func TestValidatePlayerID_ValidToken(t *testing.T) {
	keys := generateTestKeyPair(t, "key-1")
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(keys.public))
	v := newCognitoValidator(set, testIssuer)

	token := signToken(t, keys.private, "player-123", testIssuer, time.Now().Add(time.Hour))

	playerID, err := v.ValidatePlayerID(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "player-123", playerID)
}

func TestValidatePlayerID_ExpiredToken(t *testing.T) {
	keys := generateTestKeyPair(t, "key-1")
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(keys.public))
	v := newCognitoValidator(set, testIssuer)

	token := signToken(t, keys.private, "player-123", testIssuer, time.Now().Add(-time.Hour))

	_, err := v.ValidatePlayerID(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidatePlayerID_MalformedToken(t *testing.T) {
	set := jwk.NewSet()
	v := newCognitoValidator(set, testIssuer)

	_, err := v.ValidatePlayerID(context.Background(), "not-a-jwt")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidatePlayerID_WrongSigningKey(t *testing.T) {
	trusted := generateTestKeyPair(t, "key-1")
	untrusted := generateTestKeyPair(t, "key-2")
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(trusted.public))
	v := newCognitoValidator(set, testIssuer)

	token := signToken(t, untrusted.private, "player-123", testIssuer, time.Now().Add(time.Hour))

	_, err := v.ValidatePlayerID(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidatePlayerID_WrongIssuer(t *testing.T) {
	keys := generateTestKeyPair(t, "key-1")
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(keys.public))
	v := newCognitoValidator(set, testIssuer)

	token := signToken(t, keys.private, "player-123", "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_other", time.Now().Add(time.Hour))

	_, err := v.ValidatePlayerID(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidatePlayerID_MissingSubject(t *testing.T) {
	keys := generateTestKeyPair(t, "key-1")
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(keys.public))
	v := newCognitoValidator(set, testIssuer)

	tok, err := jwt.NewBuilder().
		Issuer(testIssuer).
		IssuedAt(time.Now().Add(-time.Minute)).
		Expiration(time.Now().Add(time.Hour)).
		Build()
	require.NoError(t, err)
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), keys.private))
	require.NoError(t, err)

	_, err = v.ValidatePlayerID(context.Background(), string(signed))

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestNewCognitoValidator_FetchError(t *testing.T) {
	_, err := NewCognitoValidator(context.Background(), "http://127.0.0.1:0/jwks.json", testIssuer)

	assert.Error(t, err)
}

func TestNewCognitoValidator_Success(t *testing.T) {
	keys := generateTestKeyPair(t, "key-1")
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(keys.public))
	body, err := json.Marshal(set)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	v, err := NewCognitoValidator(context.Background(), server.URL, testIssuer)
	require.NoError(t, err)

	token := signToken(t, keys.private, "player-123", testIssuer, time.Now().Add(time.Hour))
	playerID, err := v.ValidatePlayerID(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, "player-123", playerID)
}
