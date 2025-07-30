package tfutil

import (
	"context"
	"errors"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
)

type ListEngineFunc func(ctx context.Context, opts dbSDK.ListEngineOptions) ([]dbSDK.EngineDetail, error)
type ListInstanceTypeFunc func(ctx context.Context, opts dbSDK.ListInstanceTypeOptions) ([]dbSDK.InstanceType, error)
type GetEngineFunc func(ctx context.Context, id string) (*dbSDK.EngineDetail, error)
type GetInstanceTypeFunc func(ctx context.Context, id string) (*dbSDK.InstanceType, error)

func ValidateAndGetEngineID(ctx context.Context, listEngineFunc ListEngineFunc, engineName string, engineVersion string) (string, error) {
	engines, err := listEngineFunc(ctx, dbSDK.ListEngineOptions{})
	if err != nil {
		return "", err
	}
	for _, engine := range engines {
		if engine.Name == engineName && engine.Version == engineVersion {
			return engine.ID, nil
		}
	}
	return "", errors.New("engine not found")
}

func ValidateAndGetInstanceTypeID(ctx context.Context, listInstanceTypeFunc ListInstanceTypeFunc, instanceType string, engineID string, compatibleProduct string) (string, error) {
	active := "ACTIVE"
	maxLimit := 50
	instanceTypes, err := listInstanceTypeFunc(ctx, dbSDK.ListInstanceTypeOptions{
		Limit:    &maxLimit,
		Status:   &active,
		EngineID: &engineID,
	})
	if err != nil {
		return "", err
	}
	for _, instance := range instanceTypes {
		if instance.Label == instanceType && instance.CompatibleProduct == compatibleProduct {
			return instance.ID, nil
		}
	}
	return "", errors.New("instance type not found, not active or not compatible with single instance family")
}

func GetEngineNameAndVersionByID(ctx context.Context, getEngineFunc GetEngineFunc, engineID string) (name string, version string, err error) {
	engine, err := getEngineFunc(ctx, engineID)
	if err != nil {
		return "", "", err
	}
	return engine.Name, engine.Version, nil
}

func GetInstanceTypeNameByID(ctx context.Context, getInstanceTypeFunc GetInstanceTypeFunc, instanceTypeID string) (string, error) {
	instanceType, err := getInstanceTypeFunc(ctx, instanceTypeID)
	if err != nil {
		return "", err
	}
	return instanceType.Label, nil
}
