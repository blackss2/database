package database

import (
	"bytes"
	"encoding/json"
	"log"
	"sort"
	"strings"
	"text/template"
)

type Applyer interface {
	Apply(data map[string]interface{}, key string, value interface{}) bool
}

type ConfigMaker interface {
	Init(data map[string]interface{})
	TemplateString() string
	Applyer
}

type Config struct {
	driver  string
	applyer Applyer
	tp      *template.Template
	data    map[string]interface{}
}

func newConfig(driver string, ap Applyer, templateString string) *Config {
	t, err := template.New("_").Parse(templateString)
	if err != nil {
		panic(err)
	}
	c := &Config{
		driver:  driver,
		applyer: ap,
		tp:      t,
		data:    make(map[string]interface{}),
	}
	return c
}

func (c *Config) Set(key string, value interface{}) *Config {
	if !c.applyer.Apply(c.data, key, value) {
		log.Fatalln("unacceptable data", key, value)
	}
	return c
}

func (c *Config) Hash(hash map[string]interface{}) *Config {
	for k, v := range hash {
		c.applyer.Apply(c.data, k, v)
	}
	return c
}

func (c *Config) Json(data string) *Config {
	hash := make(map[string]interface{})
	err := json.Unmarshal([]byte(data), &hash)
	if err != nil {
		log.Fatalln(err, data)
	} else {
		return c.Hash(hash)
	}
	return c
}

func (c *Config) ConnectionString() string {
	var buffer bytes.Buffer
	c.tp.Execute(&buffer, c.data)
	return buffer.String()
}

func (c *Config) Open(MaxConn int64) (*Database, error) {
	db := NewDatabase(c.driver, c.ConnectionString(), MaxConn)
	err := db.Open()
	if err != nil {
		return nil, err
	}
	return db, nil
}

var gDriverHash map[string]ConfigMaker = make(map[string]ConfigMaker)

func AddDriver(driver string, cm ConfigMaker) {
	gDriverHash[strings.ToLower(driver)] = cm
}

func SupportDrivers() []string {
	list := make([]string, len(gDriverHash))
	for k, _ := range gDriverHash {
		list = append(list, k)
	}
	sort.Strings(list)
	return list
}

func Driver(driver string) *Config {
	cm, has := gDriverHash[strings.ToLower(driver)]
	if !has {
		panic("not supported driver")
	}
	c := newConfig(driver, cm, cm.TemplateString())
	cm.Init(c.data)
	return c
}
