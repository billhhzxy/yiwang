package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yiwang/internal/store"
	"yiwang/internal/tasks"
)

type API struct {
	store *store.Store
	now   func() time.Time
}

func New(store *store.Store) *API {
	return &API{
		store: store,
		now:   time.Now,
	}
}

// Register mounts routes under the provided group (e.g., /api).
func (a *API) Register(r *gin.RouterGroup) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/tasks", a.createTask)
	r.GET("/tasks", a.listTasks)
	r.GET("/tasks/ready", a.readyTasks)
	r.GET("/tasks/:id", a.getTask)
	r.PUT("/tasks/:id", a.updateTask)
	r.PATCH("/tasks/:id", a.updateTask)
	r.DELETE("/tasks/:id", a.deleteTask)
	r.POST("/tasks/:id/review", a.reviewTask)
}

type createTaskRequest struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type reviewRequest struct {
	Result string `json:"result"`
}

func (a *API) createTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json")
		return
	}
	t, err := a.store.Create(req.Question, req.Answer, a.now())
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, mapTask(t, a.now()))
}

func (a *API) listTasks(c *gin.Context) {
	now := a.now()
	all, err := a.store.All()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	filter := strings.ToLower(strings.TrimSpace(c.Query("status")))
	out := make([]taskResponse, 0, len(all))
	for _, t := range all {
		tr := mapTask(t, now)
		if filter == "" || filter == "all" || tr.Status == filter {
			out = append(out, tr)
		}
	}
	c.JSON(http.StatusOK, out)
}

func (a *API) readyTasks(c *gin.Context) {
	now := a.now()
	all, err := a.store.All()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]taskResponse, 0)
	for _, t := range all {
		tr := mapTask(t, now)
		if tr.Status == "ready" {
			out = append(out, tr)
		}
	}
	c.JSON(http.StatusOK, out)
}

func (a *API) getTask(c *gin.Context) {
	id := c.Param("id")
	t, err := a.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(c, http.StatusNotFound, err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, mapTask(t, a.now()))
}

func (a *API) updateTask(c *gin.Context) {
	id := c.Param("id")
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json")
		return
	}
	t, err := a.store.UpdateContent(id, req.Question, req.Answer, a.now())
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		} else if err != nil {
			status = http.StatusInternalServerError
		}
		writeError(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, mapTask(t, a.now()))
}

func (a *API) deleteTask(c *gin.Context) {
	id := c.Param("id")
	if err := a.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(c, http.StatusNotFound, err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) reviewTask(c *gin.Context) {
	id := c.Param("id")
	var req reviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json")
		return
	}

	result := strings.ToLower(strings.TrimSpace(req.Result))
	var remembered bool
	switch result {
	case "remembered", "remember", "ok", "done":
		remembered = true
	case "forgot", "forget", "miss":
		remembered = false
	default:
		writeError(c, http.StatusBadRequest, "result must be 'remembered' or 'forgot'")
		return
	}

	t, err := a.store.Review(id, remembered, a.now())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(c, http.StatusNotFound, err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, mapTask(t, a.now()))
}

type taskResponse struct {
	ID           string     `json:"id"`
	Question     string     `json:"question"`
	Answer       string     `json:"answer"`
	Stage        int        `json:"stage"`
	TotalStages  int        `json:"totalStages"`
	Status       string     `json:"status"`
	NextReviewAt *time.Time `json:"nextReviewAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

func mapTask(t *tasks.Task, now time.Time) taskResponse {
	var next *time.Time
	if !t.NextReviewAt.IsZero() {
		next = &t.NextReviewAt
	}
	return taskResponse{
		ID:           t.ID,
		Question:     t.Question,
		Answer:       t.Answer,
		Stage:        t.Stage,
		TotalStages:  tasks.TotalStages(),
		Status:       t.Status(now),
		NextReviewAt: next,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		CompletedAt:  t.CompletedAt,
	}
}

func writeError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}
