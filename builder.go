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
	instances         []*Instance
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
		instances:         make([]*Instance, 0),
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

func (b *Builder) Build(prototypeName string, instanceName ...string) *Instance {
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
		panic(fmt.Sprintf("could not build instance of %s %s: json error: %s", prototypeName, outline, err.Error()))
	}

	name := prototypeName
	if len(instanceName) > 0 {
		name = instanceName[0]
	}

	instance := &Instance{
		name:        name,
		baseBuilder: b,
		contents:    contents,
		tableName:   prototypeName,
		buildOnly:   proto.BuildOnly,
	}
	b.instances = append(b.instances, instance)
	return instance
}

func (b *Builder) Instance(name string, index ...int) *Instance {
	var (
		instance *Instance
		i        int
	)

	if len(index) > 0 {
		i = index[0]
	}

	for _, inst := range b.instances {
		if inst.name == name {
			if i == 0 {
				instance = inst
				break
			}
			i--
			continue
		}
	}
	if instance == nil {
		panic(fmt.Sprintf("no instance %s found", name))
	}

	return instance
}

func (b *Builder) Save() {
	for _, instance := range b.instances {
		name := instance.name
		if instance.buildOnly {
			continue
		}
		err := instance.persist(b.persistFunc, b.placeholderFormat)
		if err != nil {
			panic(fmt.Sprintf("error saving %s: %s", name, err.Error()))
		}
	}
}

func (b *Builder) Find(table, query string, instanceName ...string) []*Instance {
	var queryMap map[string]interface{}
	err := json.Unmarshal([]byte(query), &queryMap)
	if err != nil {
		panic(fmt.Sprintf("could not build query: json error: %s: %s", err.Error(), query))
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
		panic(fmt.Sprintf("could not unmarshal query result %s: %s", result, err.Error()))
	}
	instances := make([]*Instance, 0)
	name := table
	if len(instanceName) > 0 {
		name = instanceName[0]
	}
	for _, c := range contents {
		instances = append(instances, &Instance{
			name:              name,
			baseBuilder:       b,
			persistedContents: c,
			contents:          c,
			tableName:         table,
			persisted:         true,
			buildOnly:         true,
		})
	}

	b.instances = append(b.instances, instances...)
	return instances
}
