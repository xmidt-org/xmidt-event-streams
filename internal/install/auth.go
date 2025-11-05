// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/xmidt-org/xmidt-event-streams/internal/auth"

	"github.com/fogfish/opts"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AuthIn struct {
	fx.In
	Logger *zap.Logger
	Auth   Auth
}

type AuthOut struct {
	fx.Out
	Auth *auth.Auth
}

func provideAuth(in AuthIn) (AuthOut, error) {
	keyProvider := auth.NewKeyProvider(in.Auth.Jwt.KeyRefreshInterval, in.Auth.Jwt.PublicKeyUrl)

	var opts []opts.Option[auth.Auth]
	opts = append(opts,
		auth.IsBasic(in.Auth.IsBasic),
		auth.WithBasic(in.Auth.Basic),
		auth.WithJwt(in.Auth.Jwt),
		auth.WithLogger(in.Logger),
		auth.WithPublicKeyProvider(keyProvider),
	)

	auth, err := auth.New(opts)

	return AuthOut{
		Auth: auth,
	}, err
}
