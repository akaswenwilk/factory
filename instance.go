package factory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
)

type Instance struct {
	baseBuilder       *Builder
	persistedContents map[string]interface{}
	contents          map[string]interface{}
	tableName         string
	persisted         bool
}

func (i *Instance) Get(attr string) interface{} {
	val, ok := i.contents[attr]
	if !ok {
		panic(fmt.Sprintf("could not find attribute %s", attr))
	}

	return val
}

func (i *Instance) With(attr string, value interface{}) *Instance {
	newContents := make(map[string]interface{}, len(i.contents))
	for k, v := range i.contents {
		newContents[k] = v
	}
	newContents[attr] = value
	i.contents = newContents
	return i
}

func (i *Instance) Contents() string {
	jsonContents, err := json.Marshal(i.contents)
	if err != nil {
		panic(fmt.Sprintf("could not marshal contents: %v", err))
	}
	return string(jsonContents)
}

func (i *Instance) persist(save PersistFunc, placeholderFormat squirrel.PlaceholderFormat) error {
	sql, args, err := i.insert()
	if i.persisted {
		sql, args, err = i.update()
	}
	if err != nil {
		return fmt.Errorf("could not build sql: %w", err)
	}

	if err := save(context.Background(), sql, args...); err != nil {
		return fmt.Errorf("could not persist: %w", err)
	}

	i.persisted = true
	i.persistedContents = i.contents

	return nil
}

func (i *Instance) insert() (string, []interface{}, error) {
	var keys []string
	var values []interface{}
	for k, v := range i.contents {
		keys = append(keys, k)
		values = append(values, v)
	}

	return squirrel.Insert(i.tableName).Columns(keys...).Values(values...).PlaceholderFormat(i.baseBuilder.placeholderFormat).ToSql()
}

func (i *Instance) update() (string, []interface{}, error) {
	builder := squirrel.Update(i.tableName).SetMap(i.contents)

	for k, v := range i.persistedContents {
		builder = builder.Where(squirrel.Eq{k: v})
	}

	return builder.PlaceholderFormat(i.baseBuilder.placeholderFormat).ToSql()
}
