package tfutil

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
)

type DataConfig struct {
	ApiKey     string
	Env        string
	Region     string
	Keypair    KeyPairData
	CoreConfig sdk.CoreClient
}

type KeyPairData struct {
	KeyID     string
	KeySecret string
}

type findKey func(key string, out any) error

type MgcApiKey struct {
	ApiKey string
}

func (m *MgcApiKey) GetAPIKey() string {
	return m.ApiKey
}

func GetConfigsFromTags[T any](keys findKey, s T) T {
	envs := listJSONTags(s)
	for _, env := range envs {
		var value any
		if err := keys(env, &value); err == nil {
			if value != nil && !reflect.ValueOf(value).IsZero() {
				err = setField(&s, env, value)
				if err != nil {
					fmt.Printf("Error setting field %s: %v\n", env, err)
				}
			}
		}
	}
	// Server url
	if serverUrl, ok := os.LookupEnv("MGC_SERVER_URL"); ok {
		if err := setField(&s, "ServerUrl", serverUrl); err != nil {
			fmt.Printf("Error setting field ServerUrl: %v\n", err)
		}
	}
	return s
}

func listJSONTags(obj any) []string {
	t := reflect.TypeOf(obj)
	if t.Kind() != reflect.Struct {
		return nil
	}
	var tags []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tagValue := field.Tag.Get("json")
		if tagValue != "" {
			tags = append(tags, strings.Split(tagValue, ",")[0])
		}
	}
	return tags
}

func setField(obj any, name string, value any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("obj must be a pointer to a struct")
	}

	structField := v.Elem().FieldByNameFunc(func(fieldName string) bool {
		return strings.EqualFold(fieldName, name)
	})

	if !structField.IsValid() {
		return fmt.Errorf("no such field: %s in obj", name)
	}

	if !structField.CanSet() {
		return fmt.Errorf("cannot set field %s in obj", name)
	}

	valueReflet := reflect.ValueOf(value)

	if structField.Kind() == reflect.Ptr {
		if valueReflet.Kind() != reflect.Ptr {
			valPtr := reflect.New(structField.Type().Elem())
			valPtr.Elem().Set(valueReflet.Convert(structField.Type().Elem()))
			valueReflet = valPtr
		}
		structField.Set(valueReflet)
		return nil
	} else {
		if !valueReflet.Type().ConvertibleTo(structField.Type()) {
			return fmt.Errorf("cannot assign or convert value of type %s to field %s of type %s", valueReflet.Type(), name, structField.Type())
		}
		valueReflet = valueReflet.Convert(structField.Type())
		structField.Set(valueReflet)
		return nil
	}
}
