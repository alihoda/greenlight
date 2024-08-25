package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/alihoda/greenlight/internal/validator"
	"github.com/lib/pq"
)

const contextTimeout = 3 * time.Second

type Movie struct {
	Id        int64     `json:"id"`
	Title     string    `json:"title"`
	Year      int64     `json:"year,omitempty"`
	Runtime   int64     `json:"runtime,omitempty"`
	CreatedAt time.Time `json:"-"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int64     `json:"version"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more that 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 2000, "year", "must be newer than 2000")
	v.Check(movie.Year <= int64(time.Now().Year()), "year", "can not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be positive")

	v.Check(movie.Genres != nil, "genre", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genre", "must contain at least one genre")
	v.Check(len(movie.Genres) <= 5, "genre", "must not contain more that five genres")

	v.Check(validator.Unique(movie.Genres), "genre", "must not contain duplicate values")
}

type MovieModel struct {
	DB *sql.DB
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	query := `SELECT id, title, year, runtime, genres, created_at, version FROM movies WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	var movie Movie

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&movie.Id,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.CreatedAt,
		&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT id, title, year, runtime, created_at, genres, version
		FROM movies
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (genres @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0

	movies := []*Movie{}

	for rows.Next() {
		var movie Movie

		err = rows.Scan(
			&movie.Id,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			&movie.CreatedAt,
			pq.Array(&movie.Genres),
			&movie.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		movies = append(movies, &movie)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return movies, metadata, nil
}

func (m MovieModel) Insert(movie *Movie) error {
	query := `INSERT INTO movies (title, year, runtime, genres) VALUES ($1, $2, $3, $4) RETURNING id, created_at, version`

	args := []interface{}{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Id, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Update(movie *Movie) error {
	query := `
        UPDATE movies
        SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
        WHERE id = $5 AND version = $6
        RETURNING version`

	args := []interface{}{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres), movie.Id, movie.Version}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m MovieModel) Delete(id int64) error {
	query := `DELETE FROM movies WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrRecordNotFound
	}

	return nil
}
