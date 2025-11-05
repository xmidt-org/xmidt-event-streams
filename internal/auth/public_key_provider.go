// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

var AlgMap = map[string][]jwa.SignatureAlgorithm{"RSA": {jwa.RS256, jwa.RS384, jwa.RS512}, "EC": {jwa.ES256, jwa.ES384, jwa.ES512}}

type PublicKeyProvider interface {
	GetKeys(context.Context) (jwk.Set, error)
}

type PublicKey struct {
	refreshIntervalHours int
	keyUrl               string
}

func NewKeyProvider(refreshIntervalHours int, keyUrl string) PublicKeyProvider {
	return &PublicKey{
		refreshIntervalHours: refreshIntervalHours,
		keyUrl:               keyUrl,
	}
}

func (p *PublicKey) GetKeys(ctx context.Context) (jwk.Set, error) {
	cache := jwk.NewCache(ctx)
	err := cache.Register(p.keyUrl, jwk.WithPostFetcher(GetPostFetcher(ctx)), jwk.WithRefreshInterval(time.Duration(p.refreshIntervalHours)*time.Hour))
	return jwk.NewCachedSet(cache, p.keyUrl), err
}

// Comcast public keys do not have an alg field.  The underlying jwx
// library requires the alg field to work properly.
func GetPostFetcher(ctx context.Context) jwk.PostFetchFunc {
	return func(url string, keySet jwk.Set) (jwk.Set, error) {
		newKeys := jwk.NewSet()
		keys := keySet.Keys(ctx)
		for keys.Next(ctx) {
			key := keys.Pair().Value.(jwk.Key)
			if key.Algorithm().String() != "" {
				err := newKeys.AddKey(key)
				if err != nil {
					return keySet, err
				}
				continue
			}

			keyType := key.KeyType().String()

			algorithms := AlgMap[keyType]
			for _, alg := range algorithms {
				keyDup, err := key.Clone()
				if err != nil {
					return keySet, err
				}

				err = keyDup.Set(jwk.AlgorithmKey, alg)
				if err != nil {
					return keySet, err
				}

				err = newKeys.AddKey(keyDup)
				if err != nil {
					return keySet, err
				}
			}
		}
		return newKeys, nil
	}
}
