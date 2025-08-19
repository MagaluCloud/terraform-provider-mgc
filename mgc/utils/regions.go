package utils

import (
	"fmt"

	"github.com/MagaluCloud/mgc-sdk-go/client"
)

const (
	URL_PRE_PROD = "https://api.pre-prod.jaxyendy.com"
	URL_DEV_QA   = "https://api.dev-qa.jaxyendy.com"
	ENV_PROD     = "prod"
	ENV_PRE_PROD = "pre-prod"
	ENV_DEV_QA   = "dev-qa"
)

var regions = map[string]map[string]string{
	ENV_PROD: {
		"br-ne1":  client.BrNe1.String(),
		"br-mgl1": client.BrMgl1.String(),
		"br-se1":  client.BrSe1.String(),
	},
	ENV_PRE_PROD: {
		"br-ne1":  buildPreProdUrl("br-ne1"),
		"br-mgl1": buildPreProdUrl("br-mgl1"),
		"br-se1":  buildPreProdUrl("br-se1"),
	},
	ENV_DEV_QA: {
		"br-ne1":  buildQAUrl("br-ne1"),
		"br-mgl1": buildQAUrl("br-mgl1"),
		"br-se1":  buildQAUrl("br-se1"),
		"br-mc1":  buildQAUrl("br-mc1"),
	},
}
var s3Regions = map[string]map[string]string{
	ENV_PROD: {
		"br-ne1":  "br-ne1.magaluobjects.com",
		"br-mgl1": "br-se-1.magaluobjects.com",
		"br-se1":  "br-se1.magaluobjects.com",
	},
}

func RegionToUrl(region string, env string) string {
	if regions[env] == nil {
		env = ENV_PROD
	}

	if regions[env][region] == "" {
		region = "br-se1"
	}

	return regions[env][region]

}

func buildPreProdUrl(region string) string {
	return URL_PRE_PROD + "/" + region
}

func buildQAUrl(region string) string {
	if region == "" {
		return URL_DEV_QA
	}
	return URL_DEV_QA + "/" + region
}

func RegionToS3Url(region string, env string) (string, error) {
	if env == "pre-prod" {
		return "", fmt.Errorf("pre-prod is not supported for S3")
	}
	if s3Regions[env] == nil {
		env = "prod"
	}
	if s3Regions[env][region] == "" {
		region = "br-se1"
	}
	return s3Regions[env][region], nil
}
