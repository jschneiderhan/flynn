package main

import (
	"errors"

	ct "github.com/flynn/flynn/controller/types"
	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/flynn/go-sql"
)

type ProviderRepo struct {
	db *DB
}

func NewProviderRepo(db *DB) *ProviderRepo {
	return &ProviderRepo{db}
}

func (r *ProviderRepo) Add(data interface{}) error {
	p := data.(*ct.Provider)
	if p.Name == "" {
		return errors.New("controller: name must not be blank")
	}
	if p.URL == "" {
		return errors.New("controler: url must not be blank")
	}
	// TODO: validate url
	err := r.db.QueryRow("INSERT INTO providers (name, url) VALUES ($1, $2) RETURNING provider_id, created_at, updated_at", p.Name, p.URL).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	p.ID = cleanUUID(p.ID)
	return err
}

func scanProvider(s Scanner) (*ct.Provider, error) {
	p := &ct.Provider{}
	err := s.Scan(&p.ID, &p.Name, &p.URL, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		err = ErrNotFound
	}
	p.ID = cleanUUID(p.ID)
	return p, err
}

func (r *ProviderRepo) Get(id string) (interface{}, error) {
	var row Scanner
	query := "SELECT provider_id, name, url, created_at, updated_at FROM providers WHERE deleted_at IS NULL AND "
	if idPattern.MatchString(id) {
		row = r.db.QueryRow(query+"(provider_id = $1 OR name = $2) LIMIT 1", id, id)
	} else {
		row = r.db.QueryRow(query+"name = $1", id)
	}
	return scanProvider(row)
}

func (r *ProviderRepo) List() (interface{}, error) {
	rows, err := r.db.Query("SELECT provider_id, name, url, created_at, updated_at FROM providers WHERE deleted_at IS NULL ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	providers := []*ct.Provider{}
	for rows.Next() {
		provider, err := scanProvider(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}
