# database

<pre>
This databse library is driver based sql wrapper.
It use hash based config instead of connection string.
If you wanna to add anther driver, you can add driver with simple config template.
And it support useful row fetcher like a json fetcher, slice fetcher, hash fetcher.
</pre>

# import
<pre>
You can add this library and driver by import.
Here is some simple postgres example.
</pre>

~~~ go
import (
  "log"

  "github.com/blackss2/database"
  _ "github.com/blackss2/database/driver/postgres"
)

type User struct {
  Id int64           `json:"id"`
  Name string        `json:"name"`
  Password string    `json:"password"`
  TCreate  time.Time `json:"t_create"`
}

func main() {
  db, err := database.Driver("postgres").Hash(map[string]interface{}{
    "addr":    "115.68.218.221",
    "db":      "auth",
    "id":      "allbom_auth",
    "pw":      "*auth7",
    "port":    5435,
    "timeout": 5,
    "sslmode": "disable",
  }).Open(50)
  if err != nil {
    panic(err)
  }
  
  _, err = db.Query(`
    CREATE TABLE user (
      id BIGSERIAL PRIMARY KEY
      , name VARCHAR(250)
      , password VARCHAR(250)
      , t_create TIMESTAMP
    );
  `)
  if err != nil {
		panic(err)
	}

  rows, err := db.Query(`
		SELECT
			*
		FROM users
	`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
    var user User
		rows.FetchJSON(&user)
    
    log.Println(user)
	}
	return nil
}
~~~
