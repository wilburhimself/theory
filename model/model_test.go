package model

import (
	"reflect"
	"testing"
)

// Mock structs for testing
type UserWithTags struct {
	ID        int    `db:"id,pk,auto"`
	Name      string `db:"name"`
	Email     string `db:"email"`
	CreatedAt string `db:"-"` // Should be ignored
}

type UserWithTableName struct {
	ID    int
	Name  string
	Email string
}

func (u *UserWithTableName) TableName() string {
	return "custom_users"
}

func (u *UserWithTableName) PrimaryKey() *Field {
	return &Field{
		Name:       "ID",
		DBName:    "id",
		Type:      reflect.TypeOf(0),
		IsPK:      true,
		IsAuto:    true,
		IsPKHandled: true,
	}
}

type UserWithMetadataProvider struct {
	ID    int
	Name  string
	Email string
}

func (u *UserWithMetadataProvider) ExtractMetadata() (*Metadata, error) {
	return &Metadata{
		TableName: "provider_users",
		Fields: []Field{
			{Name: "ID", DBName: "id", Type: reflect.TypeOf(0), IsPK: true, IsAuto: true},
			{Name: "Name", DBName: "name", Type: reflect.TypeOf("")},
			{Name: "Email", DBName: "email", Type: reflect.TypeOf("")},
		},
	}, nil
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name    string
		model   interface{}
		want    *Metadata
		wantErr bool
	}{
		{
			name:    "nil model",
			model:   nil,
			want:    nil,
			wantErr: true,
		},
		{
			name:  "struct with tags",
			model: &UserWithTags{},
			want: &Metadata{
				TableName: "user_with_tags",
				Fields: []Field{
					{Name: "ID", DBName: "id", Type: reflect.TypeOf(0), IsPK: true, IsAuto: true},
					{Name: "Name", DBName: "name", Type: reflect.TypeOf("")},
					{Name: "Email", DBName: "email", Type: reflect.TypeOf("")},
				},
			},
			wantErr: false,
		},
		{
			name:  "struct with TableName",
			model: &UserWithTableName{},
			want: &Metadata{
				TableName: "custom_users",
				Fields: []Field{
					{Name: "ID", DBName: "id", Type: reflect.TypeOf(0), IsPK: true, IsAuto: true, IsPKHandled: true},
					{Name: "Name", DBName: "name", Type: reflect.TypeOf("")},
					{Name: "Email", DBName: "email", Type: reflect.TypeOf("")},
				},
			},
			wantErr: false,
		},
		{
			name:  "struct with MetadataProvider",
			model: &UserWithMetadataProvider{},
			want: &Metadata{
				TableName: "provider_users",
				Fields: []Field{
					{Name: "ID", DBName: "id", Type: reflect.TypeOf(0), IsPK: true, IsAuto: true},
					{Name: "Name", DBName: "name", Type: reflect.TypeOf("")},
					{Name: "Email", DBName: "email", Type: reflect.TypeOf("")},
				},
			},
			wantErr: false,
		},
		{
			name:    "non-struct type",
			model:   "not a struct",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractMetadata(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrimaryKey(t *testing.T) {
	tests := []struct {
		name    string
		model   interface{}
		wantPK  *Field
		wantErr bool
	}{
		{
			name:  "struct with tags",
			model: &UserWithTags{},
			wantPK: &Field{
				Name:   "ID",
				DBName: "id",
				Type:   reflect.TypeOf(0),
				IsPK:   true,
				IsAuto: true,
			},
			wantErr: false,
		},
		{
			name:  "struct with TableName",
			model: &UserWithTableName{},
			wantPK: &Field{
				Name:       "ID",
				DBName:    "id",
				Type:      reflect.TypeOf(0),
				IsPK:      true,
				IsAuto:    true,
				IsPKHandled: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := ExtractMetadata(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if metadata != nil {
				got := metadata.PrimaryKey()
				if !reflect.DeepEqual(got, tt.wantPK) {
					t.Errorf("PrimaryKey() = %v, want %v", got, tt.wantPK)
				}
			}
		})
	}
}

func TestTableName(t *testing.T) {
	tests := []struct {
		name      string
		model     interface{}
		wantTable string
		wantErr   bool
	}{
		{
			name:      "struct with tags",
			model:     &UserWithTags{},
			wantTable: "user_with_tags",
			wantErr:   false,
		},
		{
			name:      "struct with TableName",
			model:     &UserWithTableName{},
			wantTable: "custom_users",
			wantErr:   false,
		},
		{
			name:      "struct with MetadataProvider",
			model:     &UserWithMetadataProvider{},
			wantTable: "provider_users",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := ExtractMetadata(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if metadata != nil && metadata.TableName != tt.wantTable {
				t.Errorf("TableName = %v, want %v", metadata.TableName, tt.wantTable)
			}
		})
	}
}
