package tfutil

import (
	"errors"
	"strings"

	"github.com/MagaluCloud/mgc-sdk-go/client"
)

func RegionToUrl(region string) (string, error) {
	switch strings.ToLower(region) {
	case "br-ne1":
		return client.BrNe1.String(), nil
	case "br-se1":
		return client.BrSe1.String(), nil
	case "br-mgl1":
		return client.BrMgl1.String(), nil
	default:
		return "", errors.New("Invalid region")
	}
}
