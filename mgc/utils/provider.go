package utils

import (
	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
)

type DataConfig struct {
	ApiKey        string
	Env           string
	Region        string
	KeyPairID     string
	KeyPairSecret string
	CoreConfig    sdk.CoreClient
}
