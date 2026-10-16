package importid

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

var (
	testProject = Namespace{Name: "project", Validate: func(v string) error {
		if v == "bad" {
			return errors.New("bad project")
		}
		return nil
	}}
	testRegion = Namespace{Name: "region"}
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name            string
		raw             string
		accepted        []Namespace
		expectedID      string
		expectedProject types.String
		expectedError   string
	}{
		{
			name:            "Bare id",
			raw:             "abc-123",
			accepted:        []Namespace{testProject},
			expectedID:      "abc-123",
			expectedProject: types.StringNull(),
		},
		{
			name:            "Explicit resource without namespaces",
			raw:             "resource/abc-123",
			accepted:        []Namespace{testProject},
			expectedID:      "abc-123",
			expectedProject: types.StringNull(),
		},
		{
			name:            "Accepted namespace",
			raw:             "project/p-1/resource/abc-123",
			accepted:        []Namespace{testProject},
			expectedID:      "abc-123",
			expectedProject: types.StringValue("p-1"),
		},
		{
			name:            "Composite resource id is kept whole",
			raw:             "project/p-1/resource/cluster-1,pool-1",
			accepted:        []Namespace{testProject},
			expectedID:      "cluster-1,pool-1",
			expectedProject: types.StringValue("p-1"),
		},
		{
			name:          "Namespace followed by a bare id",
			raw:           "project/p-1/abc-123",
			accepted:      []Namespace{testProject},
			expectedError: "resource/{id}",
		},
		{
			name:          "Namespace without resource",
			raw:           "project/p-1",
			accepted:      []Namespace{testProject},
			expectedError: "resource/{id}",
		},
		{
			name:          "Resource must be the last pair",
			raw:           "resource/abc-123/project/p-1",
			accepted:      []Namespace{testProject},
			expectedError: "resource/{id}",
		},
		{
			name:          "Resource repeated before the last pair",
			raw:           "resource/x/resource/abc-123",
			accepted:      []Namespace{testProject},
			expectedError: `"resource" must be the last pair`,
		},
		{
			name:          "Leading slash",
			raw:           "/project/p-1/resource/abc-123",
			accepted:      []Namespace{testProject},
			expectedError: "resource/{id}",
		},
		{
			name:          "Namespace not accepted by this resource fails closed",
			raw:           "region/br-ne1/resource/abc-123",
			accepted:      []Namespace{testProject},
			expectedError: `unknown namespace "region"`,
		},
		{
			name:          "No namespaces accepted",
			raw:           "project/p-1/resource/abc-123",
			accepted:      nil,
			expectedError: "accepted namespaces: none",
		},
		{
			name:          "Repeated namespace",
			raw:           "project/p-1/project/p-2/resource/abc-123",
			accepted:      []Namespace{testProject},
			expectedError: `namespace "project" repeated`,
		},
		{
			name:          "Value rejected by the namespace",
			raw:           "project/bad/resource/abc-123",
			accepted:      []Namespace{testProject},
			expectedError: "bad project",
		},
		{
			name:          "Empty namespace value",
			raw:           "project//resource/abc-123",
			accepted:      []Namespace{testProject},
			expectedError: `namespace "project" has no value`,
		},
		{
			name:          "Empty id after resource",
			raw:           "project/p-1/resource/",
			accepted:      []Namespace{testProject},
			expectedError: "empty resource id",
		},
		{
			name:          "Empty import id",
			raw:           "",
			accepted:      []Namespace{testProject},
			expectedError: "empty resource id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := Parse(tc.raw, tc.accepted...)

			if tc.expectedError != "" {
				assert.ErrorContains(t, err, tc.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedID, parsed.ID)
			assert.Equal(t, tc.expectedProject, parsed.Value(testProject))
		})
	}
}

func TestParseSeveralNamespacesInAnyOrder(t *testing.T) {
	for _, raw := range []string{"project/p-1/region/br-ne1/resource/abc-123", "region/br-ne1/project/p-1/resource/abc-123"} {
		t.Run(raw, func(t *testing.T) {
			parsed, err := Parse(raw, testProject, testRegion)

			assert.NoError(t, err)
			assert.Equal(t, "abc-123", parsed.ID)
			assert.Equal(t, types.StringValue("p-1"), parsed.Value(testProject))
			assert.Equal(t, types.StringValue("br-ne1"), parsed.Value(testRegion))
		})
	}
}
