package utils

import (
	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
)

type DataConfig struct {
	ApiKey     string
	Env        string
	Region     string
	AccessKey  string
	SecretKey  string
	CoreConfig sdk.CoreClient
}
