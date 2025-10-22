package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
)

func TestValidateAndGetEngineID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	engines := []dbSDK.EngineDetail{
		{ID: "eng-1", Name: "postgres", Version: "16"},
		{ID: "eng-2", Name: "mysql", Version: "8.0"},
	}

	listOK := func(ctx context.Context, _ dbSDK.EngineFilterOptions) ([]dbSDK.EngineDetail, error) {
		return engines, nil
	}

	listErr := func(ctx context.Context, _ dbSDK.EngineFilterOptions) ([]dbSDK.EngineDetail, error) {
		return nil, errors.New("backend error")
	}

	tests := []struct {
		name         string
		fn           ListEngineFunc
		engineName   string
		engineVer    string
		expectedID   string
		expectErr    bool
		errSubstring string
	}{
		{
			name:       "found",
			fn:         listOK,
			engineName: "postgres",
			engineVer:  "16",
			expectedID: "eng-1",
		},
		{
			name:         "not found",
			fn:           listOK,
			engineName:   "postgres",
			engineVer:    "15",
			expectErr:    true,
			errSubstring: "engine not found",
		},
		{
			name:         "list error",
			fn:           listErr,
			engineName:   "postgres",
			engineVer:    "16",
			expectErr:    true,
			errSubstring: "backend error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := ValidateAndGetEngineID(ctx, tt.fn, tt.engineName, tt.engineVer)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Fatalf("expected error to contain %q, got %q", tt.errSubstring, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.expectedID {
				t.Fatalf("expected ID %q, got %q", tt.expectedID, id)
			}
		})
	}
}

func TestValidateAndGetInstanceTypeID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	instanceTypes := []dbSDK.InstanceType{
		{ID: "it-1", Label: "small", CompatibleProduct: "postgres"},
		{ID: "it-2", Label: "medium", CompatibleProduct: "mysql"},
		{ID: "it-3", Label: "small", CompatibleProduct: "mysql"},
	}

	var capturedFilter dbSDK.InstanceTypeFilterOptions
	listOK := func(ctx context.Context, filter dbSDK.InstanceTypeFilterOptions) ([]dbSDK.InstanceType, error) {
		// Capture filter for assertion
		capturedFilter = filter
		// Simulate naive filtering by engineID if provided
		if filter.EngineID != nil && *filter.EngineID == "eng-postgres" {
			return []dbSDK.InstanceType{instanceTypes[0]}, nil
		}
		return instanceTypes, nil
	}

	listErr := func(ctx context.Context, _ dbSDK.InstanceTypeFilterOptions) ([]dbSDK.InstanceType, error) {
		return nil, errors.New("list instance type failed")
	}

	tests := []struct {
		name            string
		fn              ListInstanceTypeFunc
		instanceLabel   string
		engineID        string
		compatible      string
		expectedID      string
		expectErr       bool
		errSubstring    string
		validateFilters bool
	}{
		{
			name:            "found with filter and active status",
			fn:              listOK,
			instanceLabel:   "small",
			engineID:        "eng-postgres",
			compatible:      "postgres",
			expectedID:      "it-1",
			validateFilters: true,
		},
		{
			name:          "not found wrong label",
			fn:            listOK,
			instanceLabel: "xlarge",
			engineID:      "eng-postgres",
			compatible:    "postgres",
			expectErr:     true,
			errSubstring:  "instance type not found",
		},
		{
			name:          "list error",
			fn:            listErr,
			instanceLabel: "small",
			engineID:      "eng-postgres",
			compatible:    "postgres",
			expectErr:     true,
			errSubstring:  "list instance type failed",
		},
		{
			name:          "not compatible product",
			fn:            listOK,
			instanceLabel: "small",
			engineID:      "eng-postgres",
			compatible:    "oracle",
			expectErr:     true,
			errSubstring:  "instance type not found",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := ValidateAndGetInstanceTypeID(ctx, tt.fn, tt.instanceLabel, tt.engineID, tt.compatible)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Fatalf("expected error to contain %q, got %q", tt.errSubstring, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.expectedID {
				t.Fatalf("expected ID %q, got %q", tt.expectedID, id)
			}
			if tt.validateFilters {
				if capturedFilter.Status == nil || *capturedFilter.Status != "ACTIVE" {
					t.Fatalf("expected Status ACTIVE, got %+v", capturedFilter.Status)
				}
				if capturedFilter.EngineID == nil || *capturedFilter.EngineID != tt.engineID {
					t.Fatalf("expected EngineID %q, got %+v", tt.engineID, capturedFilter.EngineID)
				}
			}
		})
	}
}

func TestGetEngineNameAndVersionByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	getOK := func(ctx context.Context, id string) (*dbSDK.EngineDetail, error) {
		if id != "eng-1" {
			return nil, fmt.Errorf("unknown engine id: %s", id)
		}
		return &dbSDK.EngineDetail{ID: id, Name: "postgres", Version: "16"}, nil
	}

	getErr := func(ctx context.Context, id string) (*dbSDK.EngineDetail, error) {
		return nil, errors.New("backend failure")
	}

	tests := []struct {
		name         string
		fn           GetEngineFunc
		id           string
		expName      string
		expVersion   string
		expectErr    bool
		errSubstring string
	}{
		{
			name:       "success",
			fn:         getOK,
			id:         "eng-1",
			expName:    "postgres",
			expVersion: "16",
		},
		{
			name:         "not found",
			fn:           getOK,
			id:           "eng-2",
			expectErr:    true,
			errSubstring: "unknown engine id",
		},
		{
			name:         "backend error",
			fn:           getErr,
			id:           "eng-1",
			expectErr:    true,
			errSubstring: "backend failure",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, version, err := GetEngineNameAndVersionByID(ctx, tt.fn, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Fatalf("expected error to contain %q, got %q", tt.errSubstring, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tt.expName {
				t.Fatalf("expected name %q, got %q", tt.expName, name)
			}
			if version != tt.expVersion {
				t.Fatalf("expected version %q, got %q", tt.expVersion, version)
			}
		})
	}
}

func TestGetInstanceTypeNameByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	getOK := func(ctx context.Context, id string) (*dbSDK.InstanceType, error) {
		switch id {
		case "it-small":
			return &dbSDK.InstanceType{ID: id, Label: "small", CompatibleProduct: "postgres"}, nil
		case "it-medium":
			return &dbSDK.InstanceType{ID: id, Label: "medium", CompatibleProduct: "postgres"}, nil
		default:
			return nil, fmt.Errorf("instance type %s not found", id)
		}
	}

	getErr := func(ctx context.Context, id string) (*dbSDK.InstanceType, error) {
		return nil, errors.New("failure fetching instance type")
	}

	tests := []struct {
		name         string
		fn           GetInstanceTypeFunc
		id           string
		expected     string
		expectErr    bool
		errSubstring string
	}{
		{
			name:     "success small",
			fn:       getOK,
			id:       "it-small",
			expected: "small",
		},
		{
			name:         "not found",
			fn:           getOK,
			id:           "it-xlarge",
			expectErr:    true,
			errSubstring: "not found",
		},
		{
			name:         "backend error",
			fn:           getErr,
			id:           "it-small",
			expectErr:    true,
			errSubstring: "failure fetching instance type",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			label, err := GetInstanceTypeNameByID(ctx, tt.fn, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Fatalf("expected error to contain %q, got %q", tt.errSubstring, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if label != tt.expected {
				t.Fatalf("expected label %q, got %q", tt.expected, label)
			}
		})
	}
}
