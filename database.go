package database

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/suapapa/go_hangul/encoding/cp949"
)

var (
	ErrNotExist = errors.New("not exist")
)

const (
	TEMP_LIMIT_CONNECTIONS = 50
)

type Database struct {
	inst        *sql.DB
	connString  string
	driver      string
	waitChan chan struct{}
	maxConn int64
	postConnect []string //TODO
	isForceUTF8 bool //TODO
	alives int64
	pingInterval time.Duration
}

func NewDatabase(driver string, connString string, maxConn int64) *Database {
	db := &Database{
		driver: driver,
		connString: connString,
		maxConn: maxConn,
		waitChan: make(chan struct{}, maxConn),
	}
	for i:=int64(0); i<maxConn; i++ {
		db.waitChan <- struct{}{}
	}
	runtime.SetFinalizer(db, func(f interface{}) {
		f.(*Database).Close()
	})
	return db
}

func (db *Database) Open() error {
	return db.executeOpen()
}

func (db *Database) executeOpen() error {
	db.Close()

	var err error
	db.inst, err = sql.Open(db.driver, db.connString)
	if err == nil && len(db.postConnect) > 0 {
		for _, v := range db.postConnect {
			db.Query(v)
		}
	}
	return err
}

func (db *Database) BeginAutoPing(interval time.Duration) {
	isNew := (db.pingInterval == 0)
	db.pingInterval = interval

	if isNew {
		go func() {
			for db.pingInterval > 0 {
				time.Sleep(db.pingInterval)
				//db.inst.Ping()
				rows, _ := db.inst.Query("SELECT 1;")
				rows.Close()
			}
		}()
	}
}

func (db *Database) EndAutoPing() {
	db.pingInterval = 0
}

func (db *Database) Close() error {
	var err error

	if db.inst != nil {
		err = db.inst.Close()
		db.inst = nil
	}
	return err
}

func (db *Database) ConnCount() int64 {
	return db.alives
}

type Rows struct {
	inst        *sql.Rows
	isFirst     bool
	isNil       bool
	Cols        []string
	isForceUTF8 bool
	callback func()
}

func (db *Database) Query(queryStr string, args ...interface{}) (*Rows, error) {
	rows := &Rows{nil, true, false, make([]string, 0, 100), db.isForceUTF8, func() {
		db.waitChan <- struct{}{}
		atomic.AddInt64(&db.alives, -1)
	}}

	if db.inst != nil {
		<- db.waitChan
		atomic.AddInt64(&db.alives, 1)

		r, err := db.inst.Query(queryStr, args...)
		if err != nil {
			if r != nil {
				r.Close()
			} else {
				db.waitChan <- struct{}{}
				atomic.AddInt64(&db.alives, -1)
			}
			return nil, err
		}
		rows.inst = r

		rows.Cols, err = rows.inst.Columns()
		if err != nil {
			return nil, err
		}

		if !rows.inst.Next() {
			rows.Close()
		} else {
			runtime.SetFinalizer(rows, func(f interface{}) {
				f.(*Rows).Close()
			})
		}
	} else {
		return nil, errors.New("db is not initialized")
	}
	return rows, nil
}

func (db *Database) MutipleQuery(queryList []string) error {
	if db.inst != nil {
		<- db.waitChan
		defer func() {
			db.waitChan <- struct{}{}
			atomic.AddInt64(&db.alives, -1)
		}()
		atomic.AddInt64(&db.alives, 1)

		tx, err := db.inst.Begin()
		if err != nil {
			db.Close()
			db.executeOpen()
			return err
		}
		splitCnt := 1000
		cnt := len(queryList) / splitCnt
		if len(queryList)%splitCnt != 0 {
			cnt++
		}
		for i := 0; i < cnt; i++ {
			inBound := i * splitCnt
			outBound := (i + 1) * splitCnt
			if outBound > len(queryList) {
				outBound = len(queryList)
			}
			_, err = tx.Exec(strings.Join(queryList[inBound:outBound], "\n"))
			if err != nil {
				tx.Rollback()
				return err
			}
		}
		err = tx.Commit()
		if err != nil {
			tx.Rollback()
			return err
		}

	} else {
		return errors.New("db is not initialized")
	}
	return nil
}

func (db *Database) InsertJSON(table string, value interface{}) (int64, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return 0, err
	}
	var hash map[string]interface{}
	err = json.Unmarshal(data, &hash)
	if err != nil {
		return 0, err
	}

	var query bytes.Buffer
	query.WriteString(fmt.Sprintf(`
		INSERT INTO %s (
	`, table))
	values := make([]interface{}, 0, len(hash))
	idx := 0
	t := reflect.ValueOf(value)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Type().Field(i)
		fv := t.Field(i)
		tagJson := strings.Split(f.Tag.Get("json"), ",")[0]
		var key string
		if len(tagJson) > 0 {
			key = tagJson
		} else {
			key = f.Name
		}
		if !strings.HasPrefix(strings.ToLower(key), "id") {
			v := fv.Interface()

			if idx > 0 {
				query.WriteString(`, `)
			}
			query.WriteString(key)

			switch f.Type.Kind() {
			case reflect.Int:
				fallthrough
			case reflect.Int8:
				fallthrough
			case reflect.Int16:
				fallthrough
			case reflect.Uint:
				fallthrough
			case reflect.Uint8:
				fallthrough
			case reflect.Uint16:
				fallthrough
			case reflect.Int32:
				fallthrough
			case reflect.Uint32:
				fallthrough
			case reflect.Uintptr: //WARN
				fallthrough
			case reflect.Uint64:
				fallthrough
			case reflect.Int64:
				fallthrough
			case reflect.Float32:
				fallthrough
			case reflect.Float64:
				fallthrough
			case reflect.Bool:
				fallthrough
			case reflect.String:
				values = append(values, v)
			default:
				switch f.Type.Name() {
				case "Time":
					values = append(values, v)
				default:
					bv, err := json.Marshal(v)
					if err != nil {
						return 0, err
					}
					values = append(values, bv)
				}
			}

			idx++
		}
	}
	query.WriteString(`
		) VALUES (
	`)
	for i, _ := range values {
		if i > 0 {
			query.WriteString(`, `)
		}
		query.WriteString(fmt.Sprintf(`$%d`, i+1))
	}
	query.WriteString(`
		)
		RETURNING id;
	`)

	rows, err := db.Query(query.String(), values...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	rows.Next()
	return rows.FetchArray()[0].(int64), nil
}

func (db *Database) UpdateJSON(table string, id int64, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var hash map[string]interface{}
	err = json.Unmarshal(data, &hash)
	if err != nil {
		return err
	}

	var query bytes.Buffer
	query.WriteString(fmt.Sprintf(`
		UPDATE %s SET
	`, table))
	values := make([]interface{}, 0, len(hash))
	idx := 0
	t := reflect.ValueOf(value)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Type().Field(i)
		fv := t.Field(i)
		tagJson := strings.Split(f.Tag.Get("json"), ",")[0]
		var key string
		if len(tagJson) > 0 {
			key = tagJson
		} else {
			key = f.Name
		}
		if !strings.HasPrefix(strings.ToLower(key), "id") {
			v := fv.Interface()

			if idx > 0 {
				query.WriteString(`, `)
			}
			query.WriteString(fmt.Sprintf(`%s=$%d`, key, idx+1))

			switch f.Type.Kind() {
			case reflect.Int:
				fallthrough
			case reflect.Int8:
				fallthrough
			case reflect.Int16:
				fallthrough
			case reflect.Uint:
				fallthrough
			case reflect.Uint8:
				fallthrough
			case reflect.Uint16:
				fallthrough
			case reflect.Int32:
				fallthrough
			case reflect.Uint32:
				fallthrough
			case reflect.Uintptr: //WARN
				fallthrough
			case reflect.Uint64:
				fallthrough
			case reflect.Int64:
				fallthrough
			case reflect.Float32:
				fallthrough
			case reflect.Float64:
				fallthrough
			case reflect.Bool:
				fallthrough
			case reflect.String:
				values = append(values, v)
			default:
				switch f.Type.Name() {
				case "Time":
					values = append(values, v)
				default:
					bv, err := json.Marshal(v)
					if err != nil {
						return err
					}
					values = append(values, bv)
				}
			}

			idx++
		}
	}
	query.WriteString(fmt.Sprintf(`
		WHERE id='%d'
	`, id))

	rows, err := db.Query(query.String(), values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (db *Database) UpdateHash(table string, id int64, hash map[string]interface{}) error {
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf(`
		UPDATE %s SET
	`, table))
	values := make([]interface{}, 0, len(hash))
	idx := 0
	for key, v := range hash {
		if !strings.HasPrefix(strings.ToLower(key), "id") {
			if idx > 0 {
				query.WriteString(`, `)
			}
			query.WriteString(fmt.Sprintf(`%s=$%d`, key, idx+1))

			ft := reflect.TypeOf(v)

			switch ft.Kind() {
			case reflect.Int:
				fallthrough
			case reflect.Int8:
				fallthrough
			case reflect.Int16:
				fallthrough
			case reflect.Uint:
				fallthrough
			case reflect.Uint8:
				fallthrough
			case reflect.Uint16:
				fallthrough
			case reflect.Int32:
				fallthrough
			case reflect.Uint32:
				fallthrough
			case reflect.Uintptr: //WARN
				fallthrough
			case reflect.Uint64:
				fallthrough
			case reflect.Int64:
				fallthrough
			case reflect.Float32:
				fallthrough
			case reflect.Float64:
				fallthrough
			case reflect.Bool:
				fallthrough
			case reflect.String:
				values = append(values, v)
			default:
				switch ft.Name() {
				case "Time":
					values = append(values, v)
				default:
					bv, err := json.Marshal(v)
					if err != nil {
						return err
					}
					values = append(values, bv)
				}
			}

			idx++
		}
	}
	query.WriteString(fmt.Sprintf(`
		WHERE id='%d'
	`, id))

	rows, err := db.Query(query.String(), values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (rows *Rows) Next() bool {
	if !rows.isNil && rows.isFirst {
		rows.isFirst = false
		return true
	}
	if rows.inst != nil {
		if !rows.inst.Next() {
			rows.Close()
		}
	} else {
		return false
	}
	return !rows.isNil
}

func (rows *Rows) FetchArray() []interface{} {
	if rows.isNil {
		return nil
	}
	if rows.inst != nil {
		rawResult := make([]*interface{}, len(rows.Cols))
		result := make([]interface{}, len(rows.Cols))

		dest := make([]interface{}, len(rows.Cols))
		for i, _ := range rawResult {
			dest[i] = &rawResult[i]
		}
		rows.inst.Scan(dest...)
		for i, raw := range rawResult {
			if raw != nil {
				v := (*raw)
				switch vs := v.(type) {
				case []byte:
					v = string(vs)
				case string:
					if rows.isForceUTF8 {
						v = UTF8(vs)
					}
				}
				result[i] = v
			} else {
				result[i] = nil
			}
		}
		return result
	} else {
		return nil
	}
}

func (rows *Rows) FetchHash() map[string]interface{} {
	if rows.isNil {
		return nil
	}
	result := make(map[string]interface{}, len(rows.Cols))

	row := rows.FetchArray()

	for i, v := range row {
		if v != nil {
			switch vs := v.(type) {
			case []byte:
				v = string(vs)
			case string:
				if rows.isForceUTF8 {
					v = UTF8(vs)
				}
			}
		}
		result[rows.Cols[i]] = v
		result[strings.ToUpper(rows.Cols[i])] = v
		result[strings.ToLower(rows.Cols[i])] = v
	}
	return result
}

func (rows *Rows) FetchJSON(value interface{}) error {
	if rows.isNil {
		return ErrNotExist
	}
	if rows.inst != nil {
		rawResult := make([]*interface{}, len(rows.Cols))
		dest := make([]interface{}, len(rows.Cols))
		for i, _ := range rawResult {
			dest[i] = &rawResult[i]
		}
		rows.inst.Scan(dest...)

		var buffer bytes.Buffer
		buffer.WriteString(`{`)
		for i, raw := range rawResult {
			if i > 0 {
				buffer.WriteString(`,`)
			}

			bkey, err := json.Marshal(rows.Cols[i])
			if err != nil {
				return err
			}
			buffer.Write(bkey)
			buffer.WriteString(`:`)

			if raw != nil {
				v := (*raw)
				switch vs := v.(type) {
				case []byte:
					buffer.Write(vs)
				case string:
					if rows.isForceUTF8 {
						vs = UTF8(vs)
					}
					bvalue, err := json.Marshal(vs)
					if err != nil {
						return err
					}
					buffer.Write(bvalue)
				default:
					bvalue, err := json.Marshal(vs)
					if err != nil {
						return err
					}
					buffer.Write(bvalue)
				}
			} else {
				buffer.WriteString("null")
			}
		}
		buffer.WriteString(`}`)
		err := json.Unmarshal(buffer.Bytes(), &value)
		if err != nil {
			return err
		}
		return nil
	} else {
		return nil
	}
}

func (rows *Rows) Close() error {
	if rows != nil {
		rows.isNil = true
		if rows.inst != nil {
			if rows.callback != nil {
				rows.callback()
			}
			err := rows.inst.Close()
			rows.inst = nil
			return err
		}
	}
	return nil
}

func (rows *Rows) IsNil() bool {
	return rows.isNil
}

func UTF8(ustr string) (str string) {
	defer func() {
		if r := recover(); r != nil {
			ustr = str
			return
		}
	}()

	bytes, err := cp949.From([]byte(ustr))
	if err != nil {
		str = ustr
	} else {
		str = string(bytes)
	}
	return
}
