package util

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
)

// decodeDefaults from field tags
func decodeDefaults(from reflect.Value, to reflect.Value) (interface{}, error) {
	toType := to.Type()
	if toType.Kind() != reflect.Struct {
		return from.Interface(), nil
	}

	for i := 0; i < toType.NumField(); i++ {
		field := toType.Field(i)
		defaultValue := field.Tag.Get("default")
		if defaultValue == "" {
			continue
		}

		switch field.Type.Kind() {
		case reflect.Bool:
			to.Field(i).SetBool(cast.ToBool(defaultValue))

		case reflect.String:
			to.Field(i).SetString(defaultValue)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			to.Field(i).SetInt(cast.ToInt64(defaultValue))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			to.Field(i).SetUint(cast.ToUint64(defaultValue))

		case reflect.Float64, reflect.Float32:
			to.Field(i).SetFloat(cast.ToFloat64(defaultValue))
		}
	}
	return from.Interface(), nil
}

// DecodeOther uses mapstructure to decode into target structure. Unused keys cause errors.
func DecodeOther(other interface{}, cc interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           cc,
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(decodeDefaults, mapstructure.StringToTimeDurationHookFunc()),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err == nil {
		err = decoder.Decode(other)
	}

	return err
}
