package main

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/MagaluCloud/mgc-sdk-go/objectstorage"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/acctest"
)

const testBucket = "tf-acctest-bucket"

type fakeObjects struct {
	objectstorage.ObjectService

	stored   map[string]string
	copied   map[string]string
	deleted  []string
	uploaded map[string]string
}

func newFakeObjects(stored map[string]string) *fakeObjects {
	return &fakeObjects{
		stored:   stored,
		copied:   map[string]string{},
		uploaded: map[string]string{},
	}
}

func (f *fakeObjects) ListAll(_ context.Context, bucket string, opts objectstorage.ObjectFilterOptions) ([]objectstorage.Object, error) {
	var objs []objectstorage.Object
	for key := range f.stored {
		if strings.HasPrefix(key, opts.Prefix) {
			objs = append(objs, objectstorage.Object{Key: key})
		}
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })
	return objs, nil
}

func (f *fakeObjects) Download(_ context.Context, bucket, key string, _ *objectstorage.DownloadOptions) ([]byte, error) {
	return []byte(f.stored[key]), nil
}

func (f *fakeObjects) Upload(_ context.Context, bucket, key string, data []byte, _ string, _ *string) error {
	f.uploaded[key] = string(data)
	return nil
}

func (f *fakeObjects) Copy(_ context.Context, src objectstorage.CopySrcConfig, dst objectstorage.CopyDstConfig) error {
	f.copied[src.ObjectKey] = dst.ObjectKey
	return nil
}

func (f *fakeObjects) Delete(_ context.Context, bucket, key string, _ *objectstorage.DeleteOptions) error {
	f.deleted = append(f.deleted, key)
	return nil
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestDownloadSet(t *testing.T) {
	t.Run("mirrors the published set into its layer", func(t *testing.T) {
		base := t.TempDir()
		// A cassette of a test that no longer exists must not survive the
		// mirror: the local layer is derived data, not a merge.
		write(t, filepath.Join(base, acctest.MainSet, "kubernetes", "TestGone.yaml"), "stale")
		objects := newFakeObjects(map[string]string{
			"cassettes/main/kubernetes/TestA.yaml": "a",
			"cassettes/main/network/TestB.yaml":    "b",
			"cassettes/main/README.txt":            "ignored",
		})

		if err := downloadSet(t.Context(), objects, testBucket, base, ""); err != nil {
			t.Fatalf("downloadSet: %v", err)
		}

		assertFile(t, filepath.Join(base, acctest.MainSet, "kubernetes", "TestA.yaml"), "a")
		assertFile(t, filepath.Join(base, acctest.MainSet, "network", "TestB.yaml"), "b")
		assertMissing(t, filepath.Join(base, acctest.MainSet, "kubernetes", "TestGone.yaml"))
		assertMissing(t, filepath.Join(base, acctest.MainSet, "README.txt"))
	})

	t.Run("sanitizes the set name", func(t *testing.T) {
		// CI passes the branch as git spells it; publish wrote it sanitized.
		base := t.TempDir()
		objects := newFakeObjects(map[string]string{
			"cassettes/feat-tags/kubernetes/TestA.yaml": "a",
		})

		if err := downloadSet(t.Context(), objects, testBucket, base, "feat/tags"); err != nil {
			t.Fatalf("downloadSet: %v", err)
		}
		assertFile(t, filepath.Join(base, "feat-tags", "kubernetes", "TestA.yaml"), "a")
	})

	t.Run("a key escaping the layer is refused", func(t *testing.T) {
		base := t.TempDir()
		objects := newFakeObjects(map[string]string{
			"cassettes/main/../../../escaped.yaml": "x",
		})

		if err := downloadSet(t.Context(), objects, testBucket, base, acctest.MainSet); err == nil {
			t.Error("want error for a key pointing outside the layer")
		}
		assertMissing(t, filepath.Join(filepath.Dir(filepath.Dir(base)), "escaped.yaml"))
	})

	t.Run("an unpublished branch set is not an error", func(t *testing.T) {
		base := t.TempDir()
		if err := downloadSet(t.Context(), newFakeObjects(nil), testBucket, base, "feat/tags"); err != nil {
			t.Fatalf("downloadSet: %v", err)
		}
	})

	t.Run("a missing main set is an error", func(t *testing.T) {
		base := t.TempDir()
		if err := downloadSet(t.Context(), newFakeObjects(nil), testBucket, base, acctest.MainSet); err == nil {
			t.Error("replay has no base layer without the main set: want error")
		}
	})
}

func TestPublishSet(t *testing.T) {
	t.Run("uploads the branch layer under the branch name", func(t *testing.T) {
		base := t.TempDir()
		write(t, filepath.Join(base, "feat-tags", "kubernetes", "TestA.yaml"), "a")
		write(t, filepath.Join(base, acctest.MainSet, "kubernetes", "TestB.yaml"), "b")
		objects := newFakeObjects(nil)

		if err := publishSet(t.Context(), objects, testBucket, base, "feat-tags"); err != nil {
			t.Fatalf("publishSet: %v", err)
		}

		want := []string{"cassettes/feat-tags/kubernetes/TestA.yaml"}
		if got := keys(objects.uploaded); !equal(got, want) {
			t.Errorf("uploaded %v, want %v", got, want)
		}
	})

	t.Run("publishing onto main is refused", func(t *testing.T) {
		base := t.TempDir()
		write(t, filepath.Join(base, acctest.MainSet, "kubernetes", "TestA.yaml"), "a")

		if err := publishSet(t.Context(), newFakeObjects(nil), testBucket, base, acctest.MainSet); err == nil {
			t.Error("only promoting a merged branch writes main: want error")
		}
	})

	t.Run("an empty layer is an error", func(t *testing.T) {
		if err := publishSet(t.Context(), newFakeObjects(nil), testBucket, t.TempDir(), "feat-tags"); err == nil {
			t.Error("want error when there is nothing recorded to publish")
		}
	})

	t.Run("credential-like content blocks the upload", func(t *testing.T) {
		base := t.TempDir()
		write(t, filepath.Join(base, "feat-tags", "kubernetes", "TestA.yaml"), "body: -----BEGIN PRIVATE KEY-----\n")
		objects := newFakeObjects(nil)

		if err := publishSet(t.Context(), objects, testBucket, base, "feat-tags"); err == nil {
			t.Error("want error: cassettes are public once published")
		}
		if len(objects.uploaded) != 0 {
			t.Errorf("nothing may be uploaded when the scan fails, got %v", keys(objects.uploaded))
		}
	})
}

func TestPromoteSet(t *testing.T) {
	t.Run("copies the branch set over main", func(t *testing.T) {
		objects := newFakeObjects(map[string]string{
			"cassettes/feat-tags/kubernetes/TestA.yaml": "new",
			"cassettes/main/kubernetes/TestA.yaml":      "old",
			"cassettes/main/kubernetes/TestB.yaml":      "untouched",
		})

		if err := promoteSet(t.Context(), objects, testBucket, "feat/tags"); err != nil {
			t.Fatalf("promoteSet: %v", err)
		}

		if got := objects.copied["cassettes/feat-tags/kubernetes/TestA.yaml"]; got != "cassettes/main/kubernetes/TestA.yaml" {
			t.Errorf("copied to %q, want the main set", got)
		}
		if len(objects.copied) != 1 {
			t.Errorf("only the branch layer is promoted, got %v", objects.copied)
		}
	})

	t.Run("promoting main onto itself is refused", func(t *testing.T) {
		if err := promoteSet(t.Context(), newFakeObjects(nil), testBucket, acctest.MainSet); err == nil {
			t.Error("want error")
		}
	})

	t.Run("a merged PR without recordings is not an error", func(t *testing.T) {
		if err := promoteSet(t.Context(), newFakeObjects(nil), testBucket, "feat/tags"); err != nil {
			t.Fatalf("promoteSet: %v", err)
		}
	})
}

func TestDeleteSet(t *testing.T) {
	t.Run("deletes every object of the set", func(t *testing.T) {
		objects := newFakeObjects(map[string]string{
			"cassettes/feat-tags/kubernetes/TestA.yaml": "a",
			"cassettes/feat-tags/network/TestB.yaml":    "b",
			"cassettes/main/kubernetes/TestA.yaml":      "keep",
		})

		if err := deleteSet(t.Context(), objects, testBucket, "feat/tags"); err != nil {
			t.Fatalf("deleteSet: %v", err)
		}

		want := []string{"cassettes/feat-tags/kubernetes/TestA.yaml", "cassettes/feat-tags/network/TestB.yaml"}
		sort.Strings(objects.deleted)
		if !equal(objects.deleted, want) {
			t.Errorf("deleted %v, want %v", objects.deleted, want)
		}
	})

	t.Run("deleting main is refused", func(t *testing.T) {
		if err := deleteSet(t.Context(), newFakeObjects(nil), testBucket, acctest.MainSet); err == nil {
			t.Error("want error")
		}
	})
}
