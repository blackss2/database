package sqlite3

import (
	"reflect"
	"strings"

	"github.com/blackss2/database"

	_ "github.com/mattn/go-sqlite3"
)

func init() {
	cm := &database.ConfigMakerBase{
		KeyHash:     make(map[string]string),
		TypeHash:    make(map[string]reflect.Kind),
		DefaultHash: make(map[string]interface{}),
		Template: strings.Replace(strings.Replace(`
			{{ .Path }}
		`, "\n", "", -1), "\t", "", -1),
	}
	cm.AddKey(reflect.String, "Path")
	database.AddDriver("sqlite3", cm)
}
