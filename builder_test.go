package factory_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"

	"github.com/akaswenwilk/factory"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder_regularSQL(t *testing.T) {
	db, err := sql.Open("postgres", "myuser:mypassword@postgres/mydb")
	require.NoError(t, err)
	defer db.Close()
	builder := factory.NewBuilder(&factory.BuilderConfig{
		PersistFunc: func(ctx context.Context, sqlStatement string, args ...any) error {
			_, err := db.ExecContext(ctx, sqlStatement, args...)
			return err
		},
	})

	require.NotNil(t, builder)
}

func TestNewBuilder_dbTrx(t *testing.T) {
	db, err := sql.Open("postgres", "myuser:mypassword@postgres/mydb")
	require.NoError(t, err)
	defer db.Close()
	trx, err := db.Begin()
	require.NoError(t, err)
	builder := factory.NewBuilder(&factory.BuilderConfig{
		PersistFunc: func(ctx context.Context, sqlStatement string, args ...any) error {
			_, err := trx.ExecContext(ctx, sqlStatement, args...)
			return err
		},
	})

	require.NotNil(t, builder)
}
