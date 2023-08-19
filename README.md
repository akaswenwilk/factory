# factory
A factory bot inspired library for golang

# Purpose
This is a library to make working with data models in test a breeze.  

# Usage

## Initializing the builder

The builder is the principle interactor for factories.  It will register prototypes and use them to generate new model instances and then persist everything.

```go
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
```

The persistence is defined as a generic function by the user to allow working with different types of database drivers with different interfaces (currently only sql type dbs are supported). For example, when creating a factory with a transaction:

```go
import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"

	"github.com/akaswenwilk/factory"
	"github.com/stretchr/testify/require"
)

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
```

## Prototypes

Once a builder is initialized, it needs to be loaded with prototypes.  These will be structs data models with prefilled values that will serve as a model for creating new models.  
Any prototypes that are loaded will NOT be saved upon calling the persister func.

### Defining and Loading Prototypes

This particular prototype will generate a new user with name "jenny" every time the default instance is generated.

```go
builder.LoadPrototype(&User{name: "jenny"})
```

Valid types are:
- string
- bool
- int

#### Prototypes with random values

Sometimes dynamic data is needed for generating new models based on a prototype.  For these values, 

```go
builder.LoadPrototype(&User{name: func() string {
    uuid.MustV4
}})
```

#### Prototypes with sequential values

#### Prototypes with custom value setter

#### Prototypes with associations

## Generating a new model instance

## Querying existing models in a database

## Persisting model instances

## Reloading model instances
