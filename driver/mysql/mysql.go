package mysql

import (
	"reflect"
	"strings"

	"github.com/blackss2/database"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	cm := &database.ConfigMakerBase{
		KeyHash:     make(map[string]string),
		TypeHash:    make(map[string]reflect.Kind),
		DefaultHash: make(map[string]interface{}),
		Template: strings.Replace(strings.Replace(`
			{{ .Id }}:{{ .Password }}@tcp({{ .Address }}:{{ .Port }})/{{ .Database }}
		`, "\n", "", -1), "\t", "", -1),
	}
	cm.AddKey(reflect.String, "Address", "Addr", "IP")
	cm.AddKey(reflect.Int, "Port")
	cm.AddKey(reflect.String, "Database", "db", "dbname")
	cm.AddKey(reflect.String, "Id", "UserId", "User")
	cm.AddKey(reflect.String, "Password", "pw", "passwd")
	cm.AddKey(reflect.Int, "Timeout")
	cm.SetDefault("Port", 3306)
	database.AddDriver("mysql", cm)
}
