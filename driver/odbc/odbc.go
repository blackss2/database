package odbc

import (
	"reflect"
	"strings"

	"github.com/blackss2/database"

	_ "github.com/alexbrainman/odbc"
)

func init() {
	cm := &database.ConfigMakerBase{
		KeyHash:  make(map[string]string),
		TypeHash: make(map[string]reflect.Kind),
		Template: strings.Replace(strings.Replace(`
			DNS={{ .Database }};UID={{ .Id }};PWD={{ .Password }}
		`, "\n", "", -1), "\t", "", -1),
	}
	cm.AddKey(reflect.String, "Database", "db", "dbname", "DNS")
	cm.AddKey(reflect.String, "Id", "UserId", "User")
	cm.AddKey(reflect.String, "Password", "pw", "passwd")
	database.AddDriver("odbc", cm)
}
