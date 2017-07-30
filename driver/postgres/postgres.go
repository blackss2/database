package postgres

import (
	"reflect"
	"strings"

	"github.com/blackss2/database"

	_ "github.com/lib/pq"
)

func init() {
	cm := &database.ConfigMakerBase{
		KeyHash:     make(map[string]string),
		TypeHash:    make(map[string]reflect.Kind),
		DefaultHash: make(map[string]interface{}),
		Template: strings.Replace(strings.Replace(`
			{{ with .Address }}host={{ . }}{{ end }}
			{{ with .Port }} port={{ . }}{{ end }}
			{{ with .Database }} dbname={{ . }}{{ end }}
			{{ with .Id }} user={{ . }}{{ end }}
			{{ with .Password }} password={{ . }}{{ end }}
			{{ with .Timeout }} connect_timeout={{ . }}{{ end }}
			{{ with .SSL }} sslmode={{ . }}{{ end }}
		`, "\n", "", -1), "\t", "", -1),
	}
	cm.AddKey(reflect.String, "Address", "Addr", "IP")
	cm.AddKey(reflect.Int, "Port")
	cm.AddKey(reflect.String, "Database", "db", "dbname")
	cm.AddKey(reflect.String, "Id", "UserId", "User")
	cm.AddKey(reflect.String, "Password", "pw", "passwd")
	cm.AddKey(reflect.Int, "Timeout")
	cm.AddKey(reflect.String, "SSL", "sslmode")
	cm.SetDefault("Port", 5432)
	database.AddDriver("postgres", cm)
}
