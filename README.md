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
builder.LoadPrototype(Prototype{TableName: "users", Outline:`{"name":"jenny"}`})
```

There is an optional attribute for the prototype: Name.  If defined, it will store the prototype under a different name when using the Build method (see below).  Otherwise the prototype is named after the table name.

#### Prototypes with random values

Sometimes dynamic data is needed for generating new models based on a prototype.  For these values, you can use built in {{variable}} syntax to replace with values. only alphanumeric characters are supported.

```go
builder.LoadPrototype(Prototype{TableName: "users", Outline:`{"id":"{{uuid}}"}`})
```

in this instance, {{uuid}} will be replaced with the result of the inbuilt uuid method from the builder which generates a uuid. Currently there are the following built in variable replacement methods that can be substituted:

- uuid - used to generate a uuid

note that the table name for a prototype will build a map inside the builder, so there can only be one prototype defined per table.  Any subsequent prototypes will overwrite the previous one.

#### Prototypes with custom value setter

if needed, a custom value generator can be loaded into the builder as well.  In all cases, the registered generator will only be called upon instance generation and once for each instance, not once per prototype loading.

```go
builder.LoadSetterFunc("randomAlphaNumeric", func() string { 
    var result []string
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

    for i:=0; i<15; i++ {
        letter := charset[rand.Intn(len(charset))-1]
        result = append(result, letter)
    }

    return strings.Join(result, "")
})

builder.LoadPrototype(Prototype{TableName: "users", Outline:`{"id":"{{uuid}}","username":"{{randomAlphaNumeric}}"}`})
```

## Generating a new model instance

after loading the prototypes, the builder can then construct instances of each resource based on the defined prototype.  These will be stored and accessible later in a map within the builder:

```go
instance := builder.Build("user", "jenny")
```

note: if there are any json errors with the outline for the prototype, this will panic. Also will panic if the table in the first argument is not a defined prototype.

the first argument is the prototype/table name, the second is a key with which the instance can be accessed:

```go
instance := builder.Instance("jenny")
```

note: if there is no instance called jenny built already in the builder, this will panic!

If desired however, the second argument can be omitted and the instance will be named the same as the table it is queuried from:

```go
builder.Build("user")
instance := builder.Instance("user")
```

instance in this example. will have the values specified in the outline, however they can be changed using the With() function.  

```go
instance.With("username", "charles")
```

This will change the outline of this specific instance to be `{"id":"<some-uuid>", "username":"charles"}`.  Also, if accessing the instance again from the builder, it will have the updated value.

If you wish to access and refer to specific values of an instance, they can be accessed via the Get() method on the instance:

```go
jennyUUID := instance.Get("id")
```

note: if there is no field found with this name, the function will panic.

## Querying existing models in a database

you can use the Find() method on the builder to query the database and load the values in an instance. The result will be an array of instances stored under the name. A predefined prototype is not required for using Find

```go
alreadyCreatedUsers := builder.Find("user", `{"username":"charles"}`, "queriedUser")
if len(alreadyCreatedUsers) > 0 {
    fmt.Println(alreadyCreatedUsers[0].Get("id"))   
}
```

In order to use the Find() method, you must provide a queryFunc similar to the persistFunc.  The queryFunc must return a string of a JSON representation of the objects returned from the db. A default func is provided that should work for most sql based dbs:

```go
	builder := factory.NewBuilder(&factory.BuilderConfig{
		QueryFunc: factory.NewQueryFunc(db),
	})
```

Once queried, the users are stored in the builder and can be accessed similarly to built instances, only with specifying the index of the instance to access:

```go
builder.Find("user", `{"username":"charles"}`, "queriedUser")
charles := builder.Instance("queriedUsers", 0)
```

if the queriedUser doesn't have an instance at the specified index, it will panic.  If the index is omitted, the first instance of the queried array is returned. Additionally, any instances queried this way are considered build only.

## Persisting model instances

None of the previous actions will actually persist anything in the database.  The method for this is Save() on the builder.  Once prototypes have been defined and instancese built and values queried, the Save() method will persist the latest state of all the instances in the builder.


Note: Save() will attempt to save each instance in the order they were built or found!

note: Save() will panic if the persistence fails

## BuildOnly

Sometimes it is useful to have access to instances to manipulate that aren't connected to the database.  For this case, be sure to add the buildOnly attribute to the prototype:

```go
builder.LoadPrototype(Prototype{TableName: "externalServiceResponse", Outline:`{"id":"{{uuid}}"}`, buildOnly: true})
responseInstance := builder.Build("externalServiceResponse", "myResponse")

contents := responseInstance.Contents()
```
