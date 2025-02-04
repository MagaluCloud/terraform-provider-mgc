package tfutil

import (
	"fmt"
	"testing"

	"github.com/MagaluCloud/mgc-sdk-go/client"
)

func TestRegionToUrl(t *testing.T) {
	type testArgs struct {
		inputRegion string
		inputEnv    string
		expectedUrl string
	}

	tests := []testArgs{
		{
			inputRegion: "br-ne1",
			inputEnv:    "prod",
			expectedUrl: client.BrNe1.String(),
		},
		{
			inputRegion: "br-se1",
			inputEnv:    "prod",
			expectedUrl: client.BrSe1.String(),
		},
		{
			inputRegion: "br-mgl1",
			inputEnv:    "prod",
			expectedUrl: client.BrMgl1.String(),
		},
		{
			inputRegion: "br-ne1",
			inputEnv:    "pre-prod",
			expectedUrl: "https://api.pre-prod.jaxyendy.com/br-ne1",
		},
		{
			inputRegion: "br-se1",
			inputEnv:    "pre-prod",
			expectedUrl: "https://api.pre-prod.jaxyendy.com/br-se1",
		},
		{
			inputRegion: "invalid-region",
			inputEnv:    "prod",
			expectedUrl: client.BrSe1.String(),
		},
		{
			inputRegion: "invalid-region",
			inputEnv:    "pre-prod",
			expectedUrl: "https://api.pre-prod.jaxyendy.com/br-se1",
		},
		{
			inputRegion: "br-ne1",
			inputEnv:    "invalid-env",
			expectedUrl: client.BrNe1.String(),
		},
		{
			inputRegion: "invalid-region",
			inputEnv:    "invalid-env",
			expectedUrl: client.BrSe1.String(),
		},
		{
			inputRegion: "",
			inputEnv:    "",
			expectedUrl: client.BrSe1.String(),
		},
		{
			inputRegion: "",
			inputEnv:    "pre-prod",
			expectedUrl: "https://api.pre-prod.jaxyendy.com/br-se1",
		},
		{
			inputRegion: "br-ne1",
			inputEnv:    "",
			expectedUrl: client.BrNe1.String(),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Region:%s,Env:%s", tt.inputRegion, tt.inputEnv), func(t *testing.T) {
			url := RegionToUrl(tt.inputRegion, tt.inputEnv)
			if url != tt.expectedUrl {
				t.Errorf("Expected URL %q, got %q", tt.expectedUrl, url)
			}
		})
	}
}