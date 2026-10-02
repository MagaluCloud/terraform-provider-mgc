package tags

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func fullPage(start int) []int {
	page := make([]int, maxPageSize)
	for i := range page {
		page[i] = start + i
	}
	return page
}

func TestListAllPages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		pages           [][]int
		expectedItems   []int
		expectedOffsets []int
	}{
		{
			name:            "no items at all",
			pages:           [][]int{{}},
			expectedItems:   nil,
			expectedOffsets: []int{0},
		},
		{
			name:            "single page shorter than the limit",
			pages:           [][]int{{1, 2, 3}},
			expectedItems:   []int{1, 2, 3},
			expectedOffsets: []int{0},
		},
		{
			name:            "short second page ends the walk",
			pages:           [][]int{fullPage(0), {1000, 1001}},
			expectedItems:   append(fullPage(0), 1000, 1001),
			expectedOffsets: []int{0, maxPageSize},
		},
		{
			name:            "full page is followed by one more request",
			pages:           [][]int{fullPage(0), {}},
			expectedItems:   fullPage(0),
			expectedOffsets: []int{0, maxPageSize},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var offsets, limits []int
			items, err := listAllPages(func(limit, offset int) ([]int, error) {
				limits = append(limits, limit)
				offsets = append(offsets, offset)

				page := offset / maxPageSize
				if page >= len(tt.pages) {
					return nil, nil
				}
				return tt.pages[page], nil
			})

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedItems, items)
			assert.Equal(t, tt.expectedOffsets, offsets)
			for _, limit := range limits {
				assert.Equal(t, maxPageSize, limit)
			}
		})
	}
}

func TestListAllPagesStopsOnError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("service unavailable")
	calls := 0

	items, err := listAllPages(func(limit, offset int) ([]int, error) {
		calls++
		if offset == 0 {
			return fullPage(0), nil
		}
		return nil, expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, items, "a partial listing must not be mistaken for the whole one")
	assert.Equal(t, 2, calls)
}
