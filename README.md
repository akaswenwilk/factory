# factory
A factory bot inspired library for golang

# Purpose
This is a library to make working with data models in test a breeze.  

# Usage

## Initializing the builder

The builder is the principle interactor for factories.  It will register prototypes and use them to generate new model instances and then persist everything.

```go
import (
    "github.com/akaswenwilk/factory"
)

func main() {
    db, err := sql.Open("mysql",
		"user:password@tcp(127.0.0.1:3306)/hello")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
    builder := factory.NewBuilder(&factory.BuilderConfig{
        PersistFunc: func(context.Context, sqlStatement string, args ...any) error {
            
        },
    })
}
```

The persistence is defined as a generic function by the user to allow working with different types of database drivers with different interfaces (currently only sql type dbs are supported). For example, when creating a factory with a transaction:



### With an existing Database

## Prototypes

### Defining and Loading Prototypes

#### Prototypes with random values

#### Prototypes with sequential values

#### Prototypes with custom value setter

#### Prototypes with associations

## Generating a new model instance

## Querying existing models in a database

## Persisting model instances

## Reloading model instances
