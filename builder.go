package factory

import "context"

type Builder struct{}

type BuilderConfig struct {
	PersistFunc func(ctx context.Context, sqlStatement string, args ...any) error
}

func NewBuilder(config *BuilderConfig) *Builder {
	return &Builder{}
}
