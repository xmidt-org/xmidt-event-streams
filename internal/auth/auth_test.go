// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fogfish/opts"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/bascule"
	"go.uber.org/zap"
)

type MockKeyProvider struct {
	mock.Mock
}

func NewMockKeyProvider() *MockKeyProvider { return &MockKeyProvider{} }

func (m *MockKeyProvider) GetKeys(ctx context.Context) (jwk.Set, error) {
	args := m.Called(ctx)
	return args.Get(0).(jwk.Set), args.Error(1)
}

type MockTokenProvider struct {
	mock.Mock
}

func NewMockTokenProvider() *MockTokenProvider { return &MockTokenProvider{} }

func (m *MockTokenProvider) GetToken() (string, error) {
	args := m.Called()
	return args.Get(0).(string), args.Error(1)
}

type AuthTestSuite struct {
	suite.Suite

	audience []string
	jwtID    string
	issuer   string

	expiration time.Time
	issuedAt   time.Time
	notBefore  time.Time
	subject    string

	capabilities     []string
	allowedResources map[string]any
	version          string

	testKey    jwk.Key
	testKeySet jwk.Set

	testJWT   jwt.Token
	signedJWT []byte

	keyProvider   *MockKeyProvider
	tokenProvider *MockTokenProvider
}

func (suite *AuthTestSuite) initializeKey() {
	var err error
	suite.testKey, err = jwk.ParseKey([]byte(`{
    "p": "7HMYtb-1dKyDp1OkdKc9WDdVMw3vtiiKDyuyRwnnwMOoYLPYxqE0CUMzw8_zXuzq7WJAmGiFd5q7oVzkbHzrtQ",
    "kty": "RSA",
    "q": "5253lCAgBLr8SR_VzzDtk_3XTHVmVIgniajMl7XM-ttrUONV86DoIm9VBx6ywEKpj5Xv3USBRNlpf8OXqWVhPw",
    "d": "G7RLbBiCkiZuepbu46G0P8J7vn5l8G6U78gcMRdEhEsaXGZz_ZnbqjW6u8KI_3akrBT__GDPf8Hx8HBNKX5T9jNQW0WtJg1XnwHOK_OJefZl2fnx-85h3tfPD4zI3m54fydce_2kDVvqTOx_XXdNJD7v5TIAgvCymQv7qvzQ0VE",
    "e": "AQAB",
    "use": "sig",
    "kid": "test",
    "qi": "a_6YlMdA9b6piRodA0MR7DwjbALlMan19wj_VkgZ8Xoilq68sGaV2CQDoAdsTW9Mjt5PpCxvJawz0AMr6LIk9w",
    "dp": "s55HgiGs_YHjzSOsBXXaEv6NuWf31l_7aMTf_DkZFYVMjpFwtotVFUg4taJuFYlSeZwux9h2s0IXEOCZIZTQFQ",
    "alg": "RS256",
    "dq": "M79xoX9laWleDAPATSnFlbfGsmP106T2IkPKK4oNIXJ6loWerHEoNrrqKkNk-LRvMZn3HmS4-uoaOuVDPi9bBQ",
    "n": "1cHjMu7H10hKxnoq3-PJT9R25bkgVX1b39faqfecC82RMcD2DkgCiKGxkCmdUzuebpmXCZuxp-rVVbjrnrI5phAdjshZlkHwV0tyJOcerXsPgu4uk_VIJgtLdvgUAtVEd8-ZF4Y9YNOAKtf2AHAoRdP0ZVH7iVWbE6qU-IN2los"
}`))

	suite.Require().NoError(err)

	suite.testKeySet = jwk.NewSet()
	err = suite.testKeySet.AddKey(suite.testKey)
	suite.Require().NoError(err)
}

func (suite *AuthTestSuite) initializeClaims() {
	suite.audience = []string{"test-audience"}
	suite.jwtID = "test-jwt"
	suite.issuer = "test-issuer"

	suite.issuedAt = time.Now().Add(-time.Second).Round(time.Second).UTC()
	suite.expiration = suite.issuedAt.Add(time.Hour)
	suite.notBefore = suite.issuedAt.Add(-time.Hour)

	suite.subject = "test-subject"

	suite.capabilities = []string{
		"x1:webpa:api:.*:all",
		"x1:webpa:api:device/.*/config\\b:all",
	}

	suite.allowedResources = make(map[string]any)
	suite.allowedResources["allowedPartners"] = []string{"comcast"}

	suite.version = "2.0"
}

func (suite *AuthTestSuite) createJWT() {
	var err error
	suite.testJWT, err = jwt.NewBuilder().
		Audience(suite.audience).
		Subject(suite.subject).
		IssuedAt(suite.issuedAt).
		Expiration(suite.expiration).
		NotBefore(suite.notBefore).
		JwtID(suite.jwtID).
		Issuer(suite.issuer).
		Claim("capabilities", suite.capabilities).
		Claim("allowedResources", suite.allowedResources).
		Claim("version", suite.version).
		Build()

	suite.Require().NoError(err)

	suite.signedJWT, err = jwt.Sign(suite.testJWT, jwt.WithKey(jwa.RS256, suite.testKey))
	suite.Require().NoError(err)
}

func (suite *AuthTestSuite) SetupSuite() {
	suite.initializeKey()
	suite.initializeClaims()
	suite.createJWT()
	suite.keyProvider = NewMockKeyProvider()
	suite.tokenProvider = NewMockTokenProvider()
}

func TestAuth(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (suite *AuthTestSuite) TestBasicAuth() {
	opts := []opts.Option[Auth]{
		IsBasic(true),
		WithBasic(Basic{
			Username: "some-username",
			Password: "some-password",
		}),
		WithLogger(zap.NewExample()),
	}

	auth, err := New(opts)
	suite.NoError(err)

	h := auth.Middleware.ThenFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t, _ := bascule.GetFrom(r)
			fmt.Println("principal:", t.Principal())
		},
	)

	goodRequest := httptest.NewRequest("GET", "/", nil)
	goodRequest.SetBasicAuth("some-username", "some-password")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, goodRequest)
	suite.Equal(http.StatusOK, response.Code)

	badRequest := httptest.NewRequest("GET", "/", nil)
	badRequest.SetBasicAuth("some-username", "some-bad-password")
	response = httptest.NewRecorder()
	h.ServeHTTP(response, badRequest)
	suite.Equal(http.StatusUnauthorized, response.Code)
}

func (suite *AuthTestSuite) TestJwtAuth() {
	suite.keyProvider.On("GetKeys", mock.Anything).Return(suite.testKeySet, nil)
	opts := []opts.Option[Auth]{
		IsBasic(false),
		WithPublicKeyProvider(suite.keyProvider),
		WithLogger(zap.NewExample()),
	}

	auth, err := New(opts)
	suite.NoError(err)

	h := auth.Middleware.ThenFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t, _ := bascule.GetFrom(r)
			fmt.Println("principal:", t.Principal())
		},
	)

	goodRequest := httptest.NewRequest("GET", "/", nil)
	goodRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(suite.signedJWT)))
	response := httptest.NewRecorder()
	h.ServeHTTP(response, goodRequest)
	suite.Equal(http.StatusOK, response.Code)

	badRequest := httptest.NewRequest("GET", "/", nil)
	badRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "some bad token"))
	response = httptest.NewRecorder()
	h.ServeHTTP(response, badRequest)
	suite.Equal(http.StatusBadRequest, response.Code)

	// no matching capabilities
	auth.Jwt.ServiceCapabilities = []string{"some-other-capability"}
	forbiddenRequest := httptest.NewRequest("GET", "/", nil)
	forbiddenRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(suite.signedJWT)))
	response = httptest.NewRecorder()
	h.ServeHTTP(response, forbiddenRequest)
	suite.Equal(http.StatusForbidden, response.Code)

	// matching capabilities
	auth.Jwt.ServiceCapabilities = []string{"x1:webpa:api:.*:all"}
	authorizedRequest := httptest.NewRequest("GET", "/", nil)
	authorizedRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(suite.signedJWT)))
	response = httptest.NewRecorder()
	h.ServeHTTP(response, authorizedRequest)
	suite.Equal(http.StatusOK, response.Code)
}
