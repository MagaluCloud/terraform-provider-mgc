package objects

import (
	"context"
	"testing"

	objSDK "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

type contentMetadataService struct{ objSDK.ObjectService }

func (contentMetadataService) Metadata(context.Context, string, string, *objSDK.MetadataOptions) (*objSDK.Object, error) {
	return &objSDK.Object{Size: 3, ContentType: "text/plain; charset=utf-8"}, nil
}

func TestObjectContentChangedSameLength(t *testing.T) {
	for _, tt := range []struct {
		name, content string
		changed       bool
	}{
		{"different bytes", "xyz", true},
		{"same bytes", "abc", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := ObjectStorageObject{Bucket: types.StringValue("contract-bucket"), Key: types.StringValue("config.txt"), Content: types.StringValue("abc"), ContentType: types.StringValue("text/plain; charset=utf-8")}
			plan := state
			plan.Content = types.StringValue(tt.content)
			r := objectStorageObjects{objects: contentMetadataService{}}
			changed, err := r.contentChanged(context.Background(), &plan, &state)
			require.NoError(t, err)
			require.Equal(t, tt.changed, changed, "equal sizes must not hide different configured bytes")
		})
	}
}
