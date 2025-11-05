// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
)

func TestExpandKeySetByAlgorithm(t *testing.T) {
	ctx := context.Background()

	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	key1, _ := jwk.FromRaw(raw)
	assert.NoError(t, key1.Set("kid", "123"))

	key2, _ := jwk.FromRaw(raw)
	err = key2.Set("kid", "124")
	assert.NoError(t, err)
	err = key2.Set("alg", jwa.RS256)
	assert.NoError(t, err)

	keySet := jwk.NewSet()

	err = keySet.AddKey(key1)
	assert.NoError(t, err)

	err = keySet.AddKey(key2)
	assert.NoError(t, err)

	f := GetPostFetcher(ctx)

	keySetWithAlg, err := f("someurl", keySet)
	assert.NoError(t, err)

	assert.Equal(t, 4, keySetWithAlg.Len())
}
