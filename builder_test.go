package factory_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"

	"github.com/akaswenwilk/factory"
	"github.com/stretchr/testify/suite"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}\b-[0-9a-fA-F]{4}\b-[0-9a-fA-F]{4}\b-[0-9a-fA-F]{4}\b-[0-9a-fA-F]{12}$`)

type BuilderSuite struct {
	suite.Suite

	db *sql.DB
}

func TestBuilderSuite(t *testing.T) {
	suite.Run(t, new(BuilderSuite))
}

func (s *BuilderSuite) SetupSuite() {
	var err error
	s.db, err = sql.Open("postgres", "user=myuser password=mypassword host=localhost dbname=mydb sslmode=disable")
	s.NoError(err)
}

func (s *BuilderSuite) TearDownSuite() {
	s.NoError(s.db.Close())
}

func (s *BuilderSuite) SetupTest() {
	_, err := s.db.Exec("Truncate users;")
	s.NoError(err)
}

func (s *BuilderSuite) newBuilder() *factory.Builder {
	return factory.NewBuilder(&factory.BuilderConfig{
		PersistFunc: func(ctx context.Context, sqlStatement string, args ...any) error {
			_, err := s.db.ExecContext(ctx, sqlStatement, args...)
			return err
		},
		QueryFunc:         factory.NewQueryFunc(s.db),
		PlaceholderFormat: squirrel.Dollar,
	})
}

func (s *BuilderSuite) TestNewBuilder_regularSQL() {
	builder := s.newBuilder()
	s.NotNil(builder)
}

func (s *BuilderSuite) TestNewBuilder_dbTrx() {
	trx, err := s.db.Begin()
	s.NoError(err)
	defer trx.Rollback()

	builder := factory.NewBuilder(&factory.BuilderConfig{
		PersistFunc: func(ctx context.Context, sqlStatement string, args ...any) error {
			_, err := trx.ExecContext(ctx, sqlStatement, args...)
			return err
		},
	})

	s.NotNil(builder)
}

func (s *BuilderSuite) TestLoadPrototypeAndCreateInstance() {
	builder := s.newBuilder()
	builder.LoadPrototype(factory.Prototype{TableName: "users", Outline: `{"id":"{{uuid}}","username":"jenny"}`})
	instance1 := builder.Build("users", "jenny1")
	instance2 := builder.Build("users", "jenny2")
	s.Equal(instance1.Get("username"), "jenny")
	s.Equal(instance2.Get("username"), "jenny")
	s.Regexp(uuidRegex, instance1.Get("id"))
	s.Regexp(uuidRegex, instance2.Get("id"))
	s.NotEqual(instance1.Get("id"), instance2.Get("id"))
}

func (s *BuilderSuite) TestLoadPrototypeAndCreateInstanceWithOtherSpecifier() {
	builder := s.newBuilder()
	builder.LoadPrototype(factory.Prototype{TableName: "users", Outline: `{"id":"{{uuid}}","username":"jenny"}`})
	instance1 := builder.Build("users", "jenny1")
	instance2 := builder.Build("users", "jenny2").With("username", "johnny")
	s.Equal(instance1.Get("username"), "jenny")
	s.Equal(instance2.Get("username"), "johnny")
	s.Equal(builder.Instance("jenny2").Get("username"), "johnny")
}

func (s *BuilderSuite) TestLoadPrototypeAndSetterFunc() {
	builder := s.newBuilder()
	builder.LoadPrototype(factory.Prototype{TableName: "users", Outline: `{"id":"{{uuid}}","username":"{{customName}}"}`})
	builder.LoadSetterFunc("customName", func() string {
		return "jimminy cricket"
	})
	s.Equal(builder.Build("users", "jim").Get("username"), "jimminy cricket")
}

func (s *BuilderSuite) TestPersistInstances() {
	builder := s.newBuilder()
	builder.LoadPrototype(factory.Prototype{TableName: "users", Outline: `{"id":"{{uuid}}","username":"jenny"}`})
	instance1 := builder.Build("users", "jenny1")
	instance2 := builder.Build("users", "jenny2")
	builder.Save()

	type user struct {
		ID       string
		Username string
	}

	for _, i := range []*factory.Instance{instance1, instance2} {
		var u user
		err := s.db.QueryRow("SELECT id, username FROM users WHERE id = $1", i.Get("id")).Scan(&u.ID, &u.Username)
		s.NoError(err)
		s.Equal(i.Get("id"), u.ID)
		s.Equal(i.Get("username"), u.Username)
	}
}

func (s *BuilderSuite) TestPersistInstancesAndQueryInstancesAndUpdate() {
	_, err := s.db.Exec("INSERT INTO users (id, username) VALUES ('123e4567-e89b-12d3-a456-426614174000', 'jenny1');")
	s.NoError(err)
	builder := s.newBuilder()
	users := builder.Find("users", "alreadyExistingUsers", `{"username":"jenny1"}`)
	s.Equal(len(users), 1)
	instance := users[0]
	s.Equal(instance.Get("id"), "123e4567-e89b-12d3-a456-426614174000")
	instance.With("username", "jenny2")
	s.Equal(instance.Get("username"), "jenny2")
	s.Equal(instance.Get("id"), "123e4567-e89b-12d3-a456-426614174000")
	builder.Save()

	type user struct {
		ID       string
		Username string
	}

	var u user

	err = s.db.QueryRow("SELECT id, username FROM users WHERE id = $1", instance.Get("id")).Scan(&u.ID, &u.Username)
	s.NoError(err)
	s.Equal(instance.Get("id"), u.ID)
	s.Equal(instance.Get("username"), u.Username)
}
