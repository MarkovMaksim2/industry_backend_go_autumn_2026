package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	var id [16]byte
	var task Task

	_, err := rand.Read(id[:])
	if err != nil {
		return task, err
	}

	title = strings.TrimSpace(title)

	if len(title) == 0 {
		return task, ErrInvalidTitle
	}
	task.ID = fmt.Sprintf("%016x", id)
	task.Title = title
	task.UpdatedAt = r.clock.Now()
	task.Done = false

	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task

	return task, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if task, ok := r.tasks[id]; !ok {
		return Task{}, ok
	} else {
		return task, ok
	}

}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]Task, 0)

	for _, item := range r.tasks {
		if item.Done == done {
			tasks = append(tasks, item)
		}
	}

	return tasks
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if item, ok := r.tasks[id]; !ok {
		return Task{}, ErrNotFound
	} else {
		if item.Done == done {
			return item, nil
		} else {
			item.Done = done
			item.UpdatedAt = r.clock.Now()
			r.tasks[item.ID] = item

			return item, nil
		}
	}
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/tasks" {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodGet {
			h.handleList(w, r)
			return
		}
		h.handleCreate(w, r)
		return
	}

	if taskID, ok := strings.CutPrefix(path, "/tasks/"); !ok || len(taskID) == 0 || strings.Contains(taskID, "/") {
		http.NotFound(w, r)
		return
	} else {
		if r.Method == http.MethodGet {
			h.handleGet(w, r, taskID)
			return
		}
		if r.Method == http.MethodPatch {
			h.handlePatch(w, r, taskID)
			return
		}
		w.Header().Set("Allow", "GET, PATCH")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title *string `json:"title"`
	}

	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, "JSON body corrupted", http.StatusBadRequest)
		return
	}

	if body.Title == nil {
		http.Error(w, "Invalid title", http.StatusBadRequest)
		return
	}

	if task, err := h.repo.Create(*body.Title); err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			http.Error(w, "Invalid title", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal", http.StatusInternalServerError)
		return
	} else {
		writeJSON(w, http.StatusCreated, task)
	}
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	if task, ok := h.repo.Get(id); ok {
		writeJSON(w, http.StatusOK, task)
		return
	}

	http.NotFound(w, r)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false

	values, ok := r.URL.Query()["done"]
	if ok {
		if len(values) != 1 {
			http.Error(w, "Bad done value", http.StatusBadRequest)
			return
		}

		if values[0] == "true" {
			done = true
		} else if values[0] == "false" {
			done = false
		} else {
			http.Error(w, "Bad done value", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
			return tasks[i].ID < tasks[j].ID
		}

		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	writeJSON(w, http.StatusOK, tasks)

}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, "Bad json", http.StatusBadRequest)
		return
	}

	if body.Done == nil {
		http.Error(w, "Invalid done", http.StatusBadRequest)
		return
	}

	if task, err := h.repo.SetDone(id, *body.Done); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	} else {
		writeJSON(w, http.StatusOK, task)
	}
}
func decodeStrictJSON(r io.Reader, v any) error {
	bodyBytes, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	bodyBytes = bytes.TrimSpace(bodyBytes)

	if len(bodyBytes) == 0 || bodyBytes[0] != '{' {
		return fmt.Errorf("Bad JSON")
	}

	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("Extra data after JSON")
	}

	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}
