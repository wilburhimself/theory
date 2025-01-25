# Theory

A lightweight and intuitive ORM for Go, inspired by the original Java version.

## Features

- Simple and intuitive API for database operations
- Support for multiple database backends (planned)
- Type-safe query building
- Migration support
- Connection pooling
- Transaction support

## Installation

```bash
go get github.com/wilburhimself/theory
```

## Usage

```go
package main

import (
    "github.com/wilburhimself/theory"
)

type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

func main() {
    // Initialize the ORM
    db, err := theory.Connect("postgres", "postgres://user:pass@localhost:5432/dbname")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Create a new user
    user := &User{
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    err = db.Create(user)
    if err != nil {
        panic(err)
    }
}
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License

[MIT](LICENSE)
