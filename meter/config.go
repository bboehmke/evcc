package meter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

type meterRegistry map[string]reflect.Type

func (r meterRegistry) Add(name string, meter api.Meter) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate meter type: %s", name))
	}
	r[name] = reflect.TypeOf(meter)
}

func (r meterRegistry) Get(name string) (reflect.Type, error) {
	t, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("meter type not registered: %s", name)
	}
	return t, nil
}

var registry meterRegistry = make(map[string]reflect.Type)

// NewFromConfig creates meter from configuration
func NewFromConfig(typ string, other map[string]interface{}) (api.Meter, error) {
	t, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, fmt.Errorf("invalid meter type: %s", typ)
	}

	v := reflect.New(t).Interface().(api.Meter)

	if err := util.DecodeOther(other, &v); err != nil {
		return nil, err
	}

	return v.Connect()
}
