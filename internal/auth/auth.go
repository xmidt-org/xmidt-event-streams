// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"net/http"

	"github.com/fogfish/opts"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/basculejwt"
	"go.uber.org/zap"
)

type Basic struct {
	Username         string
	Password         string
	UpstreamUsername string
	UpstreamPassword string
}

type Jwt struct {
	// validation
	PublicKeyUrl        string
	KeyRefreshInterval  int
	ServiceCapabilities []string

	// upstream client auth
	TokenUrl     string
	ClientId     string
	ClientSecret string
}

// options
var IsBasic = opts.ForName[Auth, bool]("isBasic")
var WithBasic = opts.ForType[Auth, Basic]()
var WithJwt = opts.ForType[Auth, Jwt]()
var WithLogger = opts.ForType[Auth, *zap.Logger]()
var WithPublicKeyProvider = opts.ForType[Auth, PublicKeyProvider]()

type Auth struct {
	logger      *zap.Logger
	isBasic     bool
	Basic       Basic
	Jwt         Jwt
	Middleware  *basculehttp.Middleware
	keyProvider PublicKeyProvider
}

func New(opt []opts.Option[Auth]) (*Auth, error) {
	var m *basculehttp.Middleware
	var err error

	auth := &Auth{}

	if err := opts.Apply(auth, opt); err != nil {
		return nil, err
	}

	if auth.isBasic {
		m, err = auth.GetBasicAuthMiddleware()
		if err != nil {
			auth.logger.Error("error creatin basic auth middleware", zap.Error(err))
		}
	} else {
		m, err = auth.GetJwtAuthMiddleware(context.Background())
		if err != nil {
			auth.logger.Error("error creatin jwt auth middleware", zap.Error(err))
		}
	}

	auth.Middleware = m

	return auth, nil
}

func (auth *Auth) GetBasicAuthMiddleware() (*basculehttp.Middleware, error) {
	tp, err := basculehttp.NewAuthorizationParser(
		basculehttp.WithBasic(),
	)
	if err != nil {
		return nil, err
	}

	m, err := basculehttp.NewMiddleware(
		basculehttp.UseAuthenticator(

			basculehttp.NewAuthenticator(
				bascule.WithTokenParsers(tp),
				bascule.WithValidators(
					bascule.AsValidator[*http.Request](
						func(token bascule.Token) error {
							if basic, ok := token.(basculehttp.BasicToken); ok && basic.UserName() == auth.Basic.Username && basic.Password() == auth.Basic.Password {
								return nil
							}

							return bascule.ErrBadCredentials
						},
					),
				),
			),
		),
	)

	return m, err
}

func (auth *Auth) GetJwtAuthMiddleware(ctx context.Context) (*basculehttp.Middleware, error) {
	keys, err := auth.keyProvider.GetKeys(ctx)
	if err != nil {
		auth.logger.Error("error getting public keys", zap.Error(err))
		return nil, err
	}

	jwtp, err := basculejwt.NewTokenParser(jwt.WithKeySet(keys))
	if err != nil {
		auth.logger.Error("error creating token parser", zap.Error(err))
		return nil, err
	}

	tp, err := basculehttp.NewAuthorizationParser(
		basculehttp.WithScheme(basculehttp.SchemeBearer, jwtp),
	)
	if err != nil {
		auth.logger.Error("error creating authoritization parser", zap.Error(err))
		return nil, err
	}

	m, err := basculehttp.NewMiddleware(
		basculehttp.UseAuthenticator(
			basculehttp.NewAuthenticator(
				bascule.WithTokenParsers(tp),
				bascule.WithValidators(
					bascule.AsValidator[*http.Request](
						func(token bascule.Token) error {
							_, ok := token.(basculejwt.Claims)
							if !ok {
								return bascule.ErrBadCredentials
							}

							capabilities, _ := bascule.GetCapabilities(token)

							if !auth.checkCapabilities(capabilities) {
								return bascule.ErrUnauthorized
							}

							return nil
						},
					),
				),
			),
		),
	)

	return m, err
}

// just checking for at least one capabiilty
func (auth *Auth) checkCapabilities(capabilities []string) bool {
	if len(auth.Jwt.ServiceCapabilities) == 0 {
		return true
	}

	for _, capability := range capabilities {
		for _, serviceCapability := range auth.Jwt.ServiceCapabilities {
			if capability == serviceCapability {
				return true
			}
		}
	}

	return false
}
