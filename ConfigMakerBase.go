package database

import (
	"math"
	"reflect"
	"strings"

	"github.com/blackss2/utility/convert"
)

type ConfigMakerBase struct {
	KeyHash      map[string]string
	RequriedHash map[string]bool
	Template     string
	TypeHash     map[string]reflect.Kind
	DefaultHash  map[string]interface{}
}

func (cm *ConfigMakerBase) Init(data map[string]interface{}) {
	for k, v := range cm.DefaultHash {
		cm.Apply(data, k, v)
	}
}

func (cm *ConfigMakerBase) SetDefault(key string, value interface{}) {
	cm.DefaultHash[key] = value
}

func (cm *ConfigMakerBase) AddKey(kind reflect.Kind, key string, morekeys ...string) {
	cm.TypeHash[key] = kind
	cm.KeyHash[strings.ToLower(key)] = key
	for _, n := range morekeys {
		cm.KeyHash[strings.ToLower(n)] = key
	}
}

func (cm *ConfigMakerBase) Apply(data map[string]interface{}, key string, value interface{}) bool {
	if k, has := cm.KeyHash[strings.ToLower(key)]; has {
		kind := cm.TypeHash[k]
		switch kind {
		case reflect.Int:
			if v := convert.IntWith(value, math.MaxInt64); v == math.MaxInt64 {
				return false
			} else {
				data[k] = v
			}
		case reflect.String:
			data[k] = convert.String(value)
		}
	}
	return true
}

func (cm *ConfigMakerBase) TemplateString() string {
	return cm.Template
}
