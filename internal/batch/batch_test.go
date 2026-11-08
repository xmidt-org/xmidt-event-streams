// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBatch(t *testing.T) {
	batchSize := 5

	items := []string{}
	batchOfItems, more := GetBatch(0, batchSize, items)
	assert.False(t, more)
	assert.Empty(t, batchOfItems)

	items = []string{"a", "b", "c", "d"}
	batchOfItems, more = GetBatch(0, batchSize, items)
	assert.False(t, more)
	assert.Len(t, batchOfItems, 4)

	items = []string{"a", "b", "c", "d", "e"}
	batchOfItems, more = GetBatch(0, batchSize, items)
	assert.False(t, more)
	assert.Len(t, batchOfItems, 5)

	items = []string{"a", "b", "c", "d", "e", "f"}
	batchOfItems, more = GetBatch(0, batchSize, items)
	assert.True(t, more)
	assert.Len(t, batchOfItems, 5)
	batchOfItems, more = GetBatch(0+batchSize, batchSize, items)
	assert.False(t, more)
	assert.Len(t, batchOfItems, 1)
}

func TestGetLargeBatch(t *testing.T) {
	batchSize := 5

	start := 0
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	batchOfItems, more := GetBatch(start, batchSize, items)
	assert.True(t, more)
	assert.Len(t, batchOfItems, 5)

	start += batchSize
	batchOfItems, more = GetBatch(start, batchSize, items)
	assert.True(t, more)
	assert.Len(t, batchOfItems, 5)

	start += batchSize
	batchOfItems, more = GetBatch(start, batchSize, items)
	assert.False(t, more)
	assert.Len(t, batchOfItems, 1)
}

func TestGetBatches100Items(t *testing.T) {
	batchSize := 25

	items := make([]string, 0, 100)

	for i := range 100 {
		items = append(items, fmt.Sprintf("%d", i))
	}

	batches := GetBatches(batchSize, items)
	assert.Len(t, batches, 4)
	assert.Len(t, batches[0], 25)
	assert.Len(t, batches[1], 25)
	assert.Len(t, batches[2], 25)
	assert.Len(t, batches[3], 25)
}

func TestGetBatchesEmptyItems(t *testing.T) {
	batchSize := 25

	items := []string{}

	batches := GetBatches(batchSize, items)
	assert.Empty(t, batches)
}

func TestGetBatchesLessThanOne(t *testing.T) {
	batchSize := 5

	items := []string{"a", "b", "c", "d"}

	batches := GetBatches(batchSize, items)
	assert.Len(t, batches, 1)
	assert.Len(t, batches[0], 4)
}

func TestGetBatchesExactlyOne(t *testing.T) {
	batchSize := 5

	items := []string{"a", "b", "c", "d", "e"}

	batches := GetBatches(batchSize, items)
	assert.Len(t, batches, 1)
	assert.Len(t, batches[0], 5)
}

func TestGetBatchesOneOver(t *testing.T) {
	batchSize := 5

	items := []string{"a", "b", "c", "d", "e", "f"}

	batches := GetBatches(batchSize, items)
	assert.Len(t, batches, 2)
	assert.Len(t, batches[0], 5)
	assert.Len(t, batches[1], 1)
}
