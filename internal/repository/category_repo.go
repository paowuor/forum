package repository

import (
	"database/sql"

	"forum/internal/models"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// GetAll returns every category, ordered alphabetically.
func (r *CategoryRepository) GetAll() ([]models.Category, error) {
	rows, err := r.db.Query(`SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// SeedDefaults inserts a starter set of categories if the table is currently empty.
// This just gives the forum something to select from out of the box; users aren't
// able to create their own categories in this implementation.
func (r *CategoryRepository) SeedDefaults() error {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []string{"General", "Technology", "Gaming", "Random"}
	for _, name := range defaults {
		if _, err := r.db.Exec(`INSERT INTO categories (name) VALUES (?)`, name); err != nil {
			return err
		}
	}
	return nil
}