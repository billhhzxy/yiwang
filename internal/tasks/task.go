package tasks

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// Task represents one Q&A item that progresses through spaced repetition.
type Task struct {
	ID           string     `json:"id"`
	Question     string     `json:"question"`
	Answer       string     `json:"answer"`
	Stage        int        `json:"stage"` // zero-based index into StageDurations
	NextReviewAt time.Time  `json:"nextReviewAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

// NewTask constructs a task at stage 0 and schedules the first review.
func NewTask(question, answer string, now time.Time) (*Task, error) {
	q := strings.TrimSpace(question)
	a := strings.TrimSpace(answer)
	if q == "" || a == "" {
		return nil, errors.New("question and answer are required")
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	t := &Task{
		ID:           id,
		Question:     q,
		Answer:       a,
		Stage:        0,
		CreatedAt:    now,
		UpdatedAt:    now,
		NextReviewAt: now.Add(StageDurations[0]),
	}
	return t, nil
}

// Status returns "done", "ready", or "pending".
func (t *Task) Status(now time.Time) string {
	if t.CompletedAt != nil || t.Stage >= TotalStages() {
		return "done"
	}
	if !t.NextReviewAt.After(now) {
		return "ready"
	}
	return "pending"
}

// MarkRemembered advances the task to the next stage or marks it completed.
func (t *Task) MarkRemembered(now time.Time) {
	if t.CompletedAt != nil {
		return
	}

	if t.Stage >= TotalStages()-1 {
		t.Stage = TotalStages()
		t.NextReviewAt = time.Time{}
		t.CompletedAt = &now
		t.UpdatedAt = now
		return
	}

	t.Stage++
	t.NextReviewAt = now.Add(StageDurations[t.Stage])
	t.UpdatedAt = now
}

// MarkForgot resets the task to the first stage.
func (t *Task) MarkForgot(now time.Time) {
	t.Stage = 0
	t.CompletedAt = nil
	t.NextReviewAt = now.Add(StageDurations[0])
	t.UpdatedAt = now
}

// UpdateContent edits the question or answer text.
func (t *Task) UpdateContent(question, answer string) error {
	q := strings.TrimSpace(question)
	a := strings.TrimSpace(answer)
	if q == "" || a == "" {
		return errors.New("question and answer are required")
	}
	t.Question = q
	t.Answer = a
	return nil
}

func generateID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
