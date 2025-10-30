package query

import (
	"strings"
	"testing"
)

func TestBuildSortClause(t *testing.T) {
	validFields := []string{"id", "title", "created_at", "updated_at", "author_id"}

	tests := []struct {
		name        string
		sorts       []string
		tableName   string
		validFields []string
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty sorts",
			sorts:       []string{},
			tableName:   "posts",
			validFields: validFields,
			want:        "",
			wantErr:     false,
		},
		{
			name:        "nil sorts",
			sorts:       nil,
			tableName:   "posts",
			validFields: validFields,
			want:        "",
			wantErr:     false,
		},
		{
			name:        "single ascending sort",
			sorts:       []string{"title"},
			tableName:   "posts",
			validFields: validFields,
			want:        "ORDER BY posts.title ASC",
			wantErr:     false,
		},
		{
			name:        "single descending sort",
			sorts:       []string{"-created_at"},
			tableName:   "posts",
			validFields: validFields,
			want:        "ORDER BY posts.created_at DESC",
			wantErr:     false,
		},
		{
			name:        "multiple mixed sorts",
			sorts:       []string{"-created_at", "title", "-author_id"},
			tableName:   "posts",
			validFields: validFields,
			want:        "ORDER BY posts.created_at DESC, posts.title ASC, posts.author_id DESC",
			wantErr:     false,
		},
		{
			name:        "all ascending sorts",
			sorts:       []string{"id", "title"},
			tableName:   "posts",
			validFields: validFields,
			want:        "ORDER BY posts.id ASC, posts.title ASC",
			wantErr:     false,
		},
		{
			name:        "all descending sorts",
			sorts:       []string{"-id", "-title"},
			tableName:   "posts",
			validFields: validFields,
			want:        "ORDER BY posts.id DESC, posts.title DESC",
			wantErr:     false,
		},
		{
			name:        "camelCase field converted to snake_case",
			sorts:       []string{"createdAt"},
			tableName:   "posts",
			validFields: validFields,
			want:        "ORDER BY posts.created_at ASC",
			wantErr:     false,
		},
		{
			name:        "camelCase descending field",
			sorts:       []string{"-createdAt"},
			tableName:   "posts",
			validFields: validFields,
			want:        "ORDER BY posts.created_at DESC",
			wantErr:     false,
		},
		{
			name:        "different table name",
			sorts:       []string{"title", "-created_at"},
			tableName:   "articles",
			validFields: validFields,
			want:        "ORDER BY articles.title ASC, articles.created_at DESC",
			wantErr:     false,
		},
		{
			name:        "invalid field",
			sorts:       []string{"invalid_field"},
			tableName:   "posts",
			validFields: validFields,
			want:        "",
			wantErr:     true,
			errContains: "invalid sort fields: invalid_field",
		},
		{
			name:        "multiple invalid fields",
			sorts:       []string{"invalid_field", "another_invalid"},
			tableName:   "posts",
			validFields: validFields,
			want:        "",
			wantErr:     true,
			errContains: "invalid sort fields:",
		},
		{
			name:        "mixed valid and invalid fields",
			sorts:       []string{"title", "invalid_field"},
			tableName:   "posts",
			validFields: validFields,
			want:        "",
			wantErr:     true,
			errContains: "invalid sort fields: invalid_field",
		},
		{
			name:        "invalid descending field",
			sorts:       []string{"-invalid_field"},
			tableName:   "posts",
			validFields: validFields,
			want:        "",
			wantErr:     true,
			errContains: "invalid sort fields: invalid_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSortClause(tt.sorts, tt.tableName, tt.validFields)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSortClause() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("BuildSortClause() error = %v, should contain %q", err, tt.errContains)
				return
			}
			if got != tt.want {
				t.Errorf("BuildSortClause() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSortFields(t *testing.T) {
	validFields := []string{"id", "title", "created_at", "updated_at", "author_id"}

	tests := []struct {
		name        string
		sorts       []string
		validFields []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "all valid fields",
			sorts:       []string{"title", "created_at"},
			validFields: validFields,
			wantErr:     false,
		},
		{
			name:        "valid fields with descending prefix",
			sorts:       []string{"-title", "-created_at"},
			validFields: validFields,
			wantErr:     false,
		},
		{
			name:        "mixed ascending and descending valid fields",
			sorts:       []string{"title", "-created_at", "id"},
			validFields: validFields,
			wantErr:     false,
		},
		{
			name:        "empty sorts",
			sorts:       []string{},
			validFields: validFields,
			wantErr:     false,
		},
		{
			name:        "single invalid field",
			sorts:       []string{"invalid"},
			validFields: validFields,
			wantErr:     true,
			errContains: "invalid sort fields: invalid",
		},
		{
			name:        "multiple invalid fields",
			sorts:       []string{"invalid1", "invalid2"},
			validFields: validFields,
			wantErr:     true,
			errContains: "invalid sort fields:",
		},
		{
			name:        "mixed valid and invalid",
			sorts:       []string{"title", "invalid", "created_at"},
			validFields: validFields,
			wantErr:     true,
			errContains: "invalid sort fields: invalid",
		},
		{
			name:        "invalid field with descending prefix",
			sorts:       []string{"-invalid"},
			validFields: validFields,
			wantErr:     true,
			errContains: "invalid sort fields: invalid",
		},
		{
			name:        "camelCase valid field",
			sorts:       []string{"createdAt"},
			validFields: validFields,
			wantErr:     false,
		},
		{
			name:        "camelCase invalid field",
			sorts:       []string{"invalidField"},
			validFields: validFields,
			wantErr:     true,
			errContains: "invalid sort fields: invalid_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSortFields(tt.sorts, tt.validFields)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSortFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("ValidateSortFields() error = %v, should contain %q", err, tt.errContains)
			}
		})
	}
}
