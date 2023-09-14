package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

const (
	uuidVar = "uuid"
)

var varReplacementRegex = regexp.MustCompile(`\{\{([a-zA-z0-9]+)\}\}`)

type (
	PersistFunc func(ctx context.Context, sqlStatement string, args ...any) error
	QueryFunc   func(ctx context.Context, sqlStatement string, args ...any) (string, error)
)

type Builder struct {
	prototypes        map[string]Prototype
	instances         map[string][]*Instance
	setterFuncs       map[string]func() string
	persistFunc       PersistFunc
	queryFunc         QueryFunc
	placeholderFormat squirrel.PlaceholderFormat
}

type BuilderConfig struct {
	PersistFunc
	QueryFunc
	squirrel.PlaceholderFormat
}

func NewBuilder(config *BuilderConfig) *Builder {
	return &Builder{
		persistFunc:       config.PersistFunc,
		queryFunc:         config.QueryFunc,
		placeholderFormat: config.PlaceholderFormat,
		prototypes:        make(map[string]Prototype),
		instances:         make(map[string][]*Instance),
		setterFuncs: map[string]func() string{
			uuidVar: func() string {
				return uuid.Must(uuid.NewV4()).String()
			},
		},
	}
}

func (b *Builder) LoadPrototype(prototype Prototype) {
	b.prototypes[prototype.TableName] = prototype
}

func (b *Builder) LoadSetterFunc(name string, f func() string) {
	b.setterFuncs[name] = f
}

func (b *Builder) Build(prototypeName, instanceName string) *Instance {
	proto, ok := b.prototypes[prototypeName]
	if !ok {
		panic(fmt.Sprintf("could not build instance of %s: no prototype found", prototypeName))
	}

	outline := proto.Outline

	vars := varReplacementRegex.FindAllStringSubmatch(outline, -1)
	for _, v := range vars {
		f, ok := b.setterFuncs[v[1]]
		if !ok {
			panic(fmt.Sprintf("could not build instance of %s: no setter function called %s found", prototypeName, v[1]))
		}
		outline = strings.ReplaceAll(outline, v[0], f())
	}

	var contents map[string]interface{}
	err := json.Unmarshal([]byte(outline), &contents)
	if err != nil {
		panic(fmt.Sprintf("could not build instance of %s: json error: %s", prototypeName, err.Error()))
	}

	instance := &Instance{
		baseBuilder: b,
		contents:    contents,
		tableName:   prototypeName,
		buildOnly:   proto.BuildOnly,
	}
	b.instances[instanceName] = []*Instance{instance}
	return instance
}

func (b *Builder) Instance(name string, index ...int) *Instance {
	instances, ok := b.instances[name]
	if !ok {
		panic(fmt.Sprintf("no instance %s found", name))
	}
	if len(index) > 0 {
		return instances[index[0]]
	}

	return instances[0]
}

func (b *Builder) Save() {
	for name, instances := range b.instances {
		for _, instance := range instances {
			if instance.buildOnly {
				continue
			}
			err := instance.persist(b.persistFunc, b.placeholderFormat)
			if err != nil {
				panic(fmt.Sprintf("error saving %s: %s", name, err.Error()))
			}
		}
	}
}

func (b *Builder) Find(table, instancesName, query string) []*Instance {
	var queryMap map[string]interface{}
	err := json.Unmarshal([]byte(query), &queryMap)
	if err != nil {
		panic(fmt.Sprintf("could not build query: json error: %s", err.Error()))
	}

	selectBuilder := squirrel.Select("*").From(table)

	for key, value := range queryMap {
		selectBuilder = selectBuilder.Where(squirrel.Eq{key: value})
	}
	selectBuilder = selectBuilder.PlaceholderFormat(b.placeholderFormat)

	sql, args, err := selectBuilder.ToSql()
	if err != nil {
		panic(fmt.Sprintf("could not build sql: %s", err.Error()))
	}

	result, err := b.queryFunc(context.Background(), sql, args...)
	if err != nil {
		panic(fmt.Sprintf("could not query %s from %s: %s", query, table, err.Error()))
	}

	var contents []map[string]interface{}
	err = json.Unmarshal([]byte(result), &contents)
	if err != nil {
		panic(fmt.Sprintf("could not unmarshal query result: %s", err.Error()))
	}
	instances := make([]*Instance, len(contents))
	for i := range instances {
		instance := &Instance{}
		instance.persisted = true
		instance.contents = contents[i]
		instance.persistedContents = contents[i]
		instance.tableName = table
		instance.baseBuilder = b
		instances[i] = instance
	}

	b.instances[instancesName] = instances
	return instances
}
