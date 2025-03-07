package tfutil

import (
	"fmt"

	"github.com/MagaluCloud/mgc-sdk-go/client"
)

const preProdUrl = "https://api.pre-prod.jaxyendy.com"

var regions = map[string]map[string]string{
	"prod": {
		"br-ne1":  client.BrNe1.String(),
		"br-mgl1": client.BrMgl1.String(),
		"br-se1":  client.BrSe1.String(),
	},
	"pre-prod": {
		"br-ne1":  buildPreProdUrl("br-ne1"),
		"br-mgl1": buildPreProdUrl("br-mgl1"),
		"br-se1":  buildPreProdUrl("br-se1"),
	},
}

func RegionToUrl(region string, env string) string {
	if regions[env] == nil {
		env = "prod"
	}
	if regions[env][region] == "" {
		region = "br-se1"
	}
	return regions[env][region]
}

func buildPreProdUrl(env string) string {
	return preProdUrl + "/" + env
}

func RegionToS3Url(region string, env string) string {
	// improve that ...
	if env == "pre-prod" {
		panic("pre-prod is not supported for S3")
	}

	var regionMap = map[string]string{
		"br-ne1":  "br-ne1",
		"br-se1":  "br-se1",
		"br-mgl1": "br-se-1",
	}

	templateUrl := "%s.magaluobjects.com"

	return fmt.Sprintf(templateUrl, regionMap[region])
}
