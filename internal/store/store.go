package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"yiwang/internal/tasks"

	_ "github.com/go-sql-driver/mysql"
)

var ErrNotFound = errors.New("task not found")

// Store manages task persistence in MySQL.
type Store struct {
	db *sql.DB
}

// New opens a MySQL-backed store and ensures schema.
func New(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.ensureTable(); err != nil {
		return nil, err
	}
	return s, nil
}

// Create adds a new task.
func (s *Store) Create(question, answer string, now time.Time) (*tasks.Task, error) {
	t, err := tasks.NewTask(question, answer, now)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO tasks (id, question, answer, stage, next_review_at, created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL)
	`, t.ID, t.Question, t.Answer, t.Stage, t.NextReviewAt, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// All returns every task.
func (s *Store) All() ([]*tasks.Task, error) {
	rows, err := s.db.Query(`
		SELECT id, question, answer, stage, next_review_at, created_at, updated_at, completed_at
		FROM tasks
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tasks.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Get returns a task by ID.
func (s *Store) Get(id string) (*tasks.Task, error) {
	row := s.db.QueryRow(`
		SELECT id, question, answer, stage, next_review_at, created_at, updated_at, completed_at
		FROM tasks
		WHERE id = ?
	`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// UpdateContent edits question/answer text.
func (s *Store) UpdateContent(id, question, answer string, now time.Time) (*tasks.Task, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
		SELECT id, question, answer, stage, next_review_at, created_at, updated_at, completed_at
		FROM tasks
		WHERE id = ?
		FOR UPDATE
	`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := t.UpdateContent(question, answer); err != nil {
		return nil, err
	}
	t.UpdatedAt = now

	if _, err := tx.Exec(`
		UPDATE tasks
		SET question = ?, answer = ?, updated_at = ?
		WHERE id = ?
	`, t.Question, t.Answer, t.UpdatedAt, t.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return t, nil
}

// Review applies a remembered/forgot result.
func (s *Store) Review(id string, remembered bool, now time.Time) (*tasks.Task, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
		SELECT id, question, answer, stage, next_review_at, created_at, updated_at, completed_at
		FROM tasks
		WHERE id = ?
		FOR UPDATE
	`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if remembered {
		t.MarkRemembered(now)
	} else {
		t.MarkForgot(now)
	}

	if _, err := tx.Exec(`
		UPDATE tasks
		SET stage = ?, next_review_at = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
	`, t.Stage, nullTime(t.NextReviewAt), nullTimePtr(t.CompletedAt), t.UpdatedAt, t.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete removes a task by ID.
func (s *Store) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ensureTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR(24) NOT NULL PRIMARY KEY,
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			stage INT NOT NULL,
			next_review_at DATETIME NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			completed_at DATETIME NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(row scanner) (*tasks.Task, error) {
	var (
		tid       string
		question  string
		answer    string
		stage     int
		next      sql.NullTime
		createdAt time.Time
		updatedAt time.Time
		completed sql.NullTime
	)
	if err := row.Scan(&tid, &question, &answer, &stage, &next, &createdAt, &updatedAt, &completed); err != nil {
		return nil, err
	}

	var nextReview time.Time
	if next.Valid {
		nextReview = next.Time
	}
	var completedAt *time.Time
	if completed.Valid {
		c := completed.Time
		completedAt = &c
	}

	return &tasks.Task{
		ID:           tid,
		Question:     question,
		Answer:       answer,
		Stage:        stage,
		NextReviewAt: nextReview,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		CompletedAt:  completedAt,
	}, nil
}

func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func nullTimePtr(t *time.Time) sql.NullTime {
	if t == nil || t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
