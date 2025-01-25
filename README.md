# Theory

A lightweight and intuitive ORM for Go, inspired by the original Java version.

## Features

- Simple and intuitive API for database operations
- Support for multiple database backends (planned)
- Type-safe query building
- Flexible model metadata definition
- Customizable table and field names
- Transaction support (planned)
- Migration support (planned)
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
)

func main() {
    // Initialize the ORM
    db, err := theory.Connect("postgres", "postgres://user:pass@localhost:5432/dbname")
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

#### Create

```go
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
}

err := db.Create(user)
if err != nil {
    panic(err)
}
```

#### Find

Find a single record:
```go
user := &User{}
err := db.Find(user, "id = ?", 1)
```

Find multiple records:
```go
var users []User
err := db.Find(&users, "age > ?", 18)
```

With query builder:
```go
var users []User
err := db.NewQuery("users").
    Select("id", "name").
    Where("age > ?", 18).
    OrderBy("name ASC").
    Limit(10).
    Offset(0).
    Find(&users)
```

#### Update

```go
user.Name = "Jane Doe"
err := db.Update(user)
```

#### Delete

```go
err := db.Delete(user)
```

### Query Building

Theory provides a fluent query builder for constructing complex queries:

```go
query := db.NewQuery("users").
    Select("id", "name", "email").
    Where("age > ?", 18).
    Where("status = ?", "active").
    OrderBy("name ASC").
    Limit(10).
    Offset(20)

var users []User
err := query.Find(&users)
```

Available query methods:
- `Select(...columns)`: Specify columns to select
- `Where(condition, ...args)`: Add WHERE conditions
- `OrderBy(expr)`: Add ORDER BY clause
- `Limit(n)`: Add LIMIT clause
- `Offset(n)`: Add OFFSET clause

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](LICENSE)
