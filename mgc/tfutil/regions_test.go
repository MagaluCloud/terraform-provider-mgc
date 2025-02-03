package tfutil

import "testing"

func TestRegions(t *testing.T) {
	type testArgs struct {
		inputRegion   string
		expectedUrl   string
		expectedError bool
	}
	ts := []testArgs{
		{
			inputRegion: "br-ne1",
			expectedUrl: "https://api.magalu.cloud/br-ne1",
		},
		{
			inputRegion: "br-se1",
			expectedUrl: "https://api.magalu.cloud/br-se1",
		},
		{
			inputRegion: "br-mgl1",
			expectedUrl: "https://api.magalu.cloud/br-se-1",
		},
		{
			inputRegion:   "br-se2",
			expectedError: true,
		},
	}

	for _, tt := range ts {
		url, err := RegionToUrl(tt.inputRegion)
		if tt.expectedError && err == nil {
			t.Errorf("Expected error but got nil")
		}
		if !tt.expectedError && err != nil {
			t.Errorf("Expected no error but got %v", err)
		}
		if url != tt.expectedUrl {
			t.Errorf("Expected url %s but got %s", tt.expectedUrl, url)
		}
	}
}
