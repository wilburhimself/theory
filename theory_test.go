package theory

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type TestUser struct {
	ID    int    `db:"id,pk,auto"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func setupTestDB(t *testing.T) (*DB, func()) {
	cfg := Config{
		Driver: "sqlite3",
		DSN:    ":memory:",  // Use in-memory mode
	}

	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Initialize migrator
	err = db.migrator.Initialize()
	if err != nil {
		db.Close()
		t.Fatalf("failed to initialize migrator: %v", err)
	}

	// Create test tables
	err = db.AutoMigrate(&TestUser{})
	if err != nil {
		db.Close()
		t.Fatalf("failed to create tables: %v", err)
	}

	// Verify table was created
	var tableName string
	row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_users'")
	err = row.Scan(&tableName)
	if err != nil {
		db.Close()
		t.Fatalf("failed to verify table creation: %v", err)
	}
	if tableName != "test_users" {
		db.Close()
		t.Fatal("test_users table was not created")
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestConnect(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Error("expected db to not be nil")
	}
}

func TestCreate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	user := &TestUser{
		Name:  "Test User",
		Email: "test@example.com",
	}

	err := db.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("expected user ID to be set after creation")
	}
}

func TestFind(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test user first
	user := &TestUser{
		Name:  "Test User",
		Email: "test@example.com",
	}
	err := db.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	var users []TestUser
	err = db.Find(context.Background(), &users, "name = ?", "Test User")
	if err != nil {
		t.Fatalf("failed to find users: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	if users[0].Name != "Test User" {
		t.Errorf("expected user name to be 'Test User', got '%s'", users[0].Name)
	}
}

func TestUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test user first
	user := &TestUser{
		Name:  "Test User",
		Email: "test@example.com",
	}
	err := db.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Update the user
	user.Name = "Updated User"
	err = db.Update(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	// Verify the update
	var updatedUser TestUser
	err = db.First(context.Background(), &updatedUser, user.ID)
	if err != nil {
		t.Fatalf("failed to get updated user: %v", err)
	}

	if updatedUser.Name != "Updated User" {
		t.Errorf("expected user name to be 'Updated User', got '%s'", updatedUser.Name)
	}
}

func TestDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test user first
	user := &TestUser{
		Name:  "Test User",
		Email: "test@example.com",
	}
	err := db.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Delete the user
	err = db.Delete(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Verify the deletion
	var deletedUser TestUser
	err = db.First(context.Background(), &deletedUser, user.ID)
	if err == nil {
		t.Error("expected error when getting deleted user")
	}
}
