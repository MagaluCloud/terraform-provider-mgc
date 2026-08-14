// Command cassettes manages the acceptance-test cassette sets in the bucket:
// download fetches a published set (default: main) into the local staging set,
// and publish credential-scans staging and uploads it under the current branch
// name.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/mgc-sdk-go/objectstorage"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/acctest"
)

const (
	envBucket    = "MGC_CASSETTE_BUCKET"
	envEndpoint  = "MGC_CASSETTE_ENDPOINT"
	envKeyID     = "MGC_CASSETTE_KEY_ID"
	envKeySecret = "MGC_CASSETTE_KEY_SECRET"

	defaultBucket = "tf-acctest-bucket"
	bucketPrefix  = "cassettes"
	mainSet       = "main"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "download":
		version := ""
		if len(os.Args) > 2 {
			version = os.Args[2]
		}
		err = download(version)
	case "publish":
		err = publish()
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

  download [set]   fetch a published cassette set (default: main) into staging
  publish          credential-scan the staging set and upload it under the branch name`)
	os.Exit(2)
}

func download(version string) error {
	if version == "" {
		version = mainSet
	}
	base, err := acctest.VCRBase()
	if err != nil {
		return err
	}
	objects, bucket, err := objectsClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	prefix := bucketPrefix + "/" + version + "/"
	objs, err := objects.ListAll(ctx, bucket, objectstorage.ObjectFilterOptions{Prefix: prefix})
	if err != nil {
		return fmt.Errorf("listing %s in bucket %s: %w", prefix, bucket, err)
	}
	if len(objs) == 0 {
		return fmt.Errorf("cassette set %q not found in bucket %s", version, bucket)
	}

	// A downloaded set seeds staging, the single set replay reads; fresh
	// recordings then overlay it before publication.
	staging := filepath.Join(base, acctest.StagingDir)
	count := 0
	for _, o := range objs {
		if !strings.HasSuffix(o.Key, ".yaml") {
			continue
		}
		data, err := objects.Download(ctx, bucket, o.Key, nil)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", o.Key, err)
		}
		target := filepath.Join(staging, filepath.FromSlash(strings.TrimPrefix(o.Key, prefix)))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		count++
	}
	fmt.Printf("downloaded %d cassettes of set %q into %s\n", count, version, staging)
	return nil
}

func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("resolving git branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "", errors.New("detached HEAD: check out a branch before materializing a cassette set")
	}
	return branch, nil
}

func publish() error {
	branch, err := currentBranch()
	if err != nil {
		return err
	}
	set, err := sanitizeSetName(branch)
	if err != nil {
		return err
	}
	base, err := acctest.VCRBase()
	if err != nil {
		return err
	}
	dir := filepath.Join(base, acctest.StagingDir)
	files, err := listYamlFiles(dir)
	if err != nil {
		return fmt.Errorf("listing staging set: %w (record or download first)", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("staging set at %s has no cassettes", dir)
	}

	// Nothing credential-shaped leaves the machine.
	for _, rel := range files {
		if err := acctest.ScanCassetteFile(filepath.Join(dir, rel)); err != nil {
			return err
		}
	}

	objects, bucket, err := objectsClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return err
		}
		key := bucketPrefix + "/" + set + "/" + filepath.ToSlash(rel)
		if err := objects.Upload(ctx, bucket, key, data, "application/yaml", nil); err != nil {
			return fmt.Errorf("uploading %s: %w", key, err)
		}
	}

	// Staging is kept: it stays the local set replay reads until the next
	// download or recording changes it.
	fmt.Printf("published %d cassettes as set %q from staging\n", len(files), set)
	return nil
}

func objectsClient() (objectstorage.ObjectService, string, error) {
	keyID, keySecret := os.Getenv(envKeyID), os.Getenv(envKeySecret)
	if keyID == "" || keySecret == "" {
		return nil, "", fmt.Errorf("%s and %s must be set (object-storage key pair with access to the cassette bucket)", envKeyID, envKeySecret)
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
