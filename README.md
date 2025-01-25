# Theory

A lightweight and intuitive ORM for Go, inspired by the original Java version.

## Features

- Simple and intuitive API for database operations
- Support for SQLite (more databases planned)
- Type-safe query building
- Flexible model metadata definition
- Customizable table and field names
- Robust error handling
- Migration support
- Connection pooling (planned)

## Installation

```bash
go get github.com/wilburhimself/theory
```

## Usage

### Connecting to a Database

```go
package main

import (
    "github.com/wilburhimself/theory"
    _ "github.com/mattn/go-sqlite3" // SQLite driver
)

func main() {
    // Initialize the ORM
    db, err := theory.Connect(theory.Config{
        Driver: "sqlite3",
        DSN:    "file:test.db?cache=shared&mode=memory",
    })
    if err != nil {
        panic(err)
    }
    defer db.Close()
}
```

### Defining Models

There are multiple ways to define your models in Theory:

#### 1. Using Struct Tags

The simplest way is to use struct tags to define your model's metadata:

```go
type User struct {
    ID    int    `db:"id,pk,auto"`   // Primary key with auto-increment
    Name  string `db:"name"`         // Regular field
    Email string `db:"email,null"`   // Nullable field
}
```

Available struct tag options:
- `pk`: Marks the field as a primary key
- `auto`: Enables auto-increment for numeric primary keys
- `null`: Allows the field to be NULL in the database
- `db:"-"`: Excludes the field from database operations

#### 2. Implementing the Model Interface

For more control over your model's metadata, you can implement the Model interface:

```go
type User struct {
    ID    int
    Name  string
    Email string
}

func (u *User) TableName() string {
    return "custom_users"  // Custom table name
}

func (u *User) PrimaryKey() *model.Field {
    return &model.Field{
        Name:   "ID",
        DBName: "id",
        Type:   reflect.TypeOf(0),
        IsPK:   true,
        IsAuto: true,
    }
}
```

#### 3. Implementing the MetadataProvider Interface

For complete control over your model's metadata:

```go
func (u *User) ExtractMetadata() (*model.Metadata, error) {
    return &model.Metadata{
        TableName: "custom_users",
        Fields: []model.Field{
            {Name: "ID", DBName: "id", Type: reflect.TypeOf(0), IsPK: true, IsAuto: true},
            {Name: "Name", DBName: "name", Type: reflect.TypeOf("")},
            {Name: "Email", DBName: "email", Type: reflect.TypeOf(""), IsNull: true},
        },
    }, nil
}
```

### CRUD Operations

All CRUD operations require a context:

#### Create

```go
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
}

err := db.Create(context.Background(), user)
if err != nil {
    panic(err)
}
```

#### Find

Find a single record:
```go
user := &User{}
err := db.First(context.Background(), user, 1) // Find by primary key
if err == theory.ErrRecordNotFound {
    // Handle not found case
}
```

Find multiple records:
```go
var users []User
err := db.Find(context.Background(), &users, "age > ?", 18)
```

#### Update

```go
user.Name = "Jane Doe"
err := db.Update(context.Background(), user)
```

#### Delete

```go
err := db.Delete(context.Background(), user)
```

### Database Migrations

Theory provides a robust migration system that supports both automatic migrations based on models and manual migrations for more complex schema changes.

#### Auto Migrations

The simplest way to manage your database schema is using auto-migrations:

```go
type User struct {
    ID        int       `db:"id,pk,auto"`
    Name      string    `db:"name"`
    Email     string    `db:"email,null"`
    CreatedAt time.Time `db:"created_at"`
}

// Create or update tables based on models
err := db.AutoMigrate(&User{})
if err != nil {
    panic(err)
}
```

#### Manual Migrations

For more complex schema changes, you can create manual migrations:

```go
func createUserMigration() *migration.Migration {
    m := migration.NewMigration("create_users_table")

    // Define up operations
    m.Up = []migration.Operation{
        &migration.CreateTable{
            Name: "users",
            Columns: []migration.Column{
                {Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
                {Name: "name", Type: "TEXT", IsNull: false},
                {Name: "email", Type: "TEXT", IsNull: true},
            },
            ForeignKeys: []migration.ForeignKey{
                {
                    Columns:      []string{"team_id"},
                    RefTable:     "teams",
                    RefColumns:   []string{"id"},
                    OnDelete:     "CASCADE",
                    OnUpdate:     "CASCADE",
                },
            },
            Indexes: []migration.Index{
                {
                    Name:    "idx_users_email",
                    Columns: []string{"email"},
                    Unique:  true,
                },
            },
        },
    }

    // Define down operations
    m.Down = []migration.Operation{
        &migration.DropTable{Name: "users"},
    }

    return m
}
```

#### Running Migrations

Theory provides several ways to run migrations:

```go
// Create a new migrator
migrator := migration.NewMigrator(db)

// Add migrations
migrator.Add(createUserMigration())
migrator.Add(createTeamMigration())

// Run all pending migrations in a transaction
err := migrator.Up()
if err != nil {
    panic(err)
}

// Roll back the last batch of migrations
err = migrator.Down()
if err != nil {
    panic(err)
}

// Check migration status
status, err := migrator.Status()
if err != nil {
    panic(err)
}
for _, s := range status {
    fmt.Printf("Migration: %s, Applied: %v, Batch: %d\n", 
        s.Migration.Name, 
        s.Applied != nil,
        s.Batch)
}
```

#### Migration Features

Theory's migration system supports:

- **Foreign Keys**: Define relationships between tables with ON DELETE and ON UPDATE actions
- **Indexes**: Create and drop indexes, including unique constraints
- **Batch Migrations**: Run multiple migrations as a single transaction
- **Rollback Support**: Easily roll back migrations by batch
- **Migration Status**: Track which migrations have been applied and when
- **Error Handling**: Robust error handling with descriptive messages
- **Validation**: Type validation for SQLite column types

#### Migration Operations

Available migration operations:

- `CreateTable`: Create a new table with columns, foreign keys, and indexes
- `DropTable`: Remove an existing table
- `AddColumn`: Add a new column to an existing table
- `ModifyColumn`: Modify an existing column's properties
- `CreateIndex`: Create a new index on specified columns
- `DropIndex`: Remove an existing index
- `AddForeignKey`: Add a new foreign key constraint

## Error Handling

Theory provides clear error types for common scenarios:

```go
// Record not found
if err == theory.ErrRecordNotFound {
    // Handle not found case
}

// Other errors
if err != nil {
    // Handle other errors
}
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](LICENSE)
