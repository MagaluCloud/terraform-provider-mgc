package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/mgc-sdk-go/objectstorage"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/acctest"
)

const (
	envBucket    = "MGC_CASSETTE_BUCKET"
	envEndpoint  = "MGC_CASSETTE_ENDPOINT"
	envKeyID     = "MGC_CASSETTE_KEY_PAIR_ID"
	envKeySecret = "MGC_CASSETTE_KEY_PAIR_SECRET"

	defaultBucket = "terraform-vcr"
	bucketPrefix  = "cassettes"
	yamlExt       = ".yaml"
)

func main() {
	arg := ""
	if len(os.Args) > 2 {
		arg = os.Args[2]
	}
	if len(os.Args) < 2 {
		usage()
	}

	var err error
	switch os.Args[1] {
	case "download":
		err = download(arg)
	case "publish":
		err = publish()
	case "promote":
		err = withSet(arg, promoteSet)
	case "delete":
		err = withSet(arg, deleteSet)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cassettes:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: cassettes <command>

  download [set]   mirror a published set (default: main) into its local layer
  publish          credential-scan the branch layer and upload it under the branch name
  promote <set>    copy a merged branch set over the published main set
  delete <set>     remove a published branch set from the bucket`)
	os.Exit(2)
}

func run(cmd func(context.Context, objectstorage.ObjectService, string) error) error {
	objects, bucket, err := objectsClient()
	if err != nil {
		return err
	}
	return cmd(context.Background(), objects, bucket)
}

func withSet(set string, cmd func(context.Context, objectstorage.ObjectService, string, string) error) error {
	if set == "" {
		usage()
	}
	return run(func(ctx context.Context, objects objectstorage.ObjectService, bucket string) error {
		return cmd(ctx, objects, bucket, set)
	})
}

func download(set string) error {
	base, err := acctest.VCRBase()
	if err != nil {
		return err
	}
	return run(func(ctx context.Context, objects objectstorage.ObjectService, bucket string) error {
		return downloadSet(ctx, objects, bucket, base, set)
	})
}

func publish() error {
	set, err := acctest.CurrentSet()
	if err != nil {
		return err
	}
	base, err := acctest.VCRBase()
	if err != nil {
		return err
	}
	return run(func(ctx context.Context, objects objectstorage.ObjectService, bucket string) error {
		return publishSet(ctx, objects, bucket, base, set)
	})
}

func setPrefix(set string) string { return bucketPrefix + "/" + set + "/" }

func downloadSet(ctx context.Context, objects objectstorage.ObjectService, bucket, base, set string) error {
	if set == "" {
		set = acctest.MainSet
	}
	set, err := acctest.SetName(set)
	if err != nil {
		return err
	}

	prefix := setPrefix(set)
	objs, err := listSet(ctx, objects, bucket, set)
	if err != nil {
		return err
	}
	if len(objs) == 0 {
		if set == acctest.MainSet {
			return fmt.Errorf("cassette set %q not found in bucket %s: replay has no base layer without it", set, bucket)
		}
		fmt.Printf("no cassettes published for set %q; nothing to download\n", set)
		return nil
	}

	dir := filepath.Join(base, set)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clearing local set %s: %w", dir, err)
	}
	for _, o := range objs {
		target, err := layerPath(dir, strings.TrimPrefix(o.Key, prefix))
		if err != nil {
			return err
		}
		data, err := objects.Download(ctx, bucket, o.Key, nil)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", o.Key, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("downloaded %d cassettes of set %q into %s\n", len(objs), set, dir)
	return nil
}

func layerPath(dir, key string) (string, error) {
	target := filepath.Join(dir, filepath.FromSlash(key))
	if target != dir && !strings.HasPrefix(target, dir+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing key %q: it resolves outside %s", key, dir)
	}
	return target, nil
}

func publishSet(ctx context.Context, objects objectstorage.ObjectService, bucket, base, set string) error {
	set, err := acctest.SetName(set)
	if err != nil {
		return err
	}
	if set == acctest.MainSet {
		return fmt.Errorf("refusing to publish onto %q: the published set is only written by promoting a merged branch — record from a branch", acctest.MainSet)
	}

	dir := filepath.Join(base, set)
	files, err := listYamlFiles(dir)
	if err != nil {
		return fmt.Errorf("listing local set: %w (record or download first)", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("branch layer at %s has no cassettes", dir)
	}

	// Nothing credential-shaped leaves the machine, and the whole set is
	// scanned before the first upload.
	for _, rel := range files {
		if err := acctest.ScanCassetteFile(filepath.Join(dir, rel)); err != nil {
			return err
		}
	}

	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return err
		}
		key := setPrefix(set) + filepath.ToSlash(rel)
		if err := objects.Upload(ctx, bucket, key, data, "application/yaml", nil); err != nil {
			return fmt.Errorf("uploading %s: %w", key, err)
		}
	}
	fmt.Printf("published %d cassettes as set %q\n", len(files), set)
	return nil
}

func promoteSet(ctx context.Context, objects objectstorage.ObjectService, bucket, set string) error {
	set, err := acctest.SetName(set)
	if err != nil {
		return err
	}
	if set == acctest.MainSet {
		return fmt.Errorf("refusing to promote %q onto itself", acctest.MainSet)
	}

	objs, err := listSet(ctx, objects, bucket, set)
	if err != nil {
		return err
	}
	if len(objs) == 0 {
		fmt.Printf("set %q has no cassettes; %q is unchanged\n", set, acctest.MainSet)
		return nil
	}

	prefix := setPrefix(set)
	for _, o := range objs {
		dst := setPrefix(acctest.MainSet) + strings.TrimPrefix(o.Key, prefix)
		src := objectstorage.CopySrcConfig{BucketName: bucket, ObjectKey: o.Key}
		if err := objects.Copy(ctx, src, objectstorage.CopyDstConfig{BucketName: bucket, ObjectKey: dst}); err != nil {
			return fmt.Errorf("copying %s to %s: %w", o.Key, dst, err)
		}
	}
	fmt.Printf("promoted %d cassettes of set %q into %q\n", len(objs), set, acctest.MainSet)
	return nil
}

func deleteSet(ctx context.Context, objects objectstorage.ObjectService, bucket, set string) error {
	set, err := acctest.SetName(set)
	if err != nil {
		return err
	}
	if set == acctest.MainSet {
		return fmt.Errorf("refusing to delete the published %q set", acctest.MainSet)
	}

	objs, err := listSet(ctx, objects, bucket, set)
	if err != nil {
		return err
	}
	for _, o := range objs {
		if err := objects.Delete(ctx, bucket, o.Key, nil); err != nil {
			return fmt.Errorf("deleting %s: %w", o.Key, err)
		}
	}
	fmt.Printf("deleted %d cassettes of set %q\n", len(objs), set)
	return nil
}

func listSet(ctx context.Context, objects objectstorage.ObjectService, bucket, set string) ([]objectstorage.Object, error) {
	prefix := setPrefix(set)
	objs, err := objects.ListAll(ctx, bucket, objectstorage.ObjectFilterOptions{Prefix: prefix})
	if err != nil {
		return nil, fmt.Errorf("listing %s in bucket %s: %w", prefix, bucket, err)
	}
	yamls := make([]objectstorage.Object, 0, len(objs))
	for _, o := range objs {
		if strings.HasSuffix(o.Key, yamlExt) {
			yamls = append(yamls, o)
		}
	}
	return yamls, nil
}

func objectsClient() (objectstorage.ObjectService, string, error) {
	keyID, keySecret := os.Getenv(envKeyID), os.Getenv(envKeySecret)
	if keyID == "" || keySecret == "" {
		return nil, "", fmt.Errorf("%s and %s must be set: the id and secret of an object-storage key pair whose owner the bucket policy allows — the provider's %s is not used here",
			envKeyID, envKeySecret, acctest.EnvAPIKey)
	}
	endpoint := objectstorage.BrNe1
	if e := os.Getenv(envEndpoint); e != "" {
		endpoint = objectstorage.Endpoint(e)
	}
	bucket := os.Getenv(envBucket)
	if bucket == "" {
		bucket = defaultBucket
	}
	client, err := objectstorage.NewWithEndpoint(sdk.NewMgcClient(sdk.WithAPIKey("cassettes-tool")), endpoint, keyID, keySecret)
	if err != nil {
		return nil, "", err
	}
	return client.Objects(), bucket, nil
}
