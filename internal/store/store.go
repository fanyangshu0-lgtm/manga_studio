package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"manga-drama-studio/internal/domain"
)

var ErrNotFound = errors.New("not found")

type state struct {
	Projects  map[string]domain.Project  `json:"projects"`
	Workflows map[string]domain.Workflow `json:"workflows"`
	Providers map[string]domain.Provider `json:"providers"`
	Runs      map[string]domain.Run      `json:"runs"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	state state
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, state: emptyState()}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create data directory: %w", err)
		}
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read data file: %w", err)
	}
	if err := json.Unmarshal(data, &s.state); err != nil {
		return nil, fmt.Errorf("decode data file: %w", err)
	}
	s.normalize()
	return s, nil
}

func emptyState() state {
	return state{
		Projects: map[string]domain.Project{}, Workflows: map[string]domain.Workflow{},
		Providers: map[string]domain.Provider{}, Runs: map[string]domain.Run{},
	}
}

func (s *Store) normalize() {
	if s.state.Projects == nil {
		s.state.Projects = map[string]domain.Project{}
	}
	if s.state.Workflows == nil {
		s.state.Workflows = map[string]domain.Workflow{}
	}
	if s.state.Providers == nil {
		s.state.Providers = map[string]domain.Provider{}
	}
	if s.state.Runs == nil {
		s.state.Runs = map[string]domain.Run{}
	}
}

func (s *Store) persistLocked() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write store: %w", err)
	}
	if err := os.Rename(temporary, s.path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace store: %w", err)
	}
	return nil
}

func clone[T any](value T) T {
	data, _ := json.Marshal(value)
	var result T
	_ = json.Unmarshal(data, &result)
	return result
}

func (s *Store) CreateProject(ctx context.Context, project domain.Project, workflow domain.Workflow) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Projects[project.ID] = clone(project)
	s.state.Workflows[project.ID] = clone(workflow)
	return s.persistLocked()
}

func (s *Store) ListProjects(ctx context.Context) ([]domain.Project, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Project, 0, len(s.state.Projects))
	for _, item := range s.state.Projects {
		items = append(items, clone(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *Store) Project(ctx context.Context, id string) (domain.Project, error) {
	if err := ctx.Err(); err != nil {
		return domain.Project{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.state.Projects[id]
	if !ok {
		return domain.Project{}, ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) Workflow(ctx context.Context, projectID string) (domain.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return domain.Workflow{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.state.Workflows[projectID]
	if !ok {
		return domain.Workflow{}, ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) SaveWorkflow(ctx context.Context, workflow domain.Workflow) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	project, ok := s.state.Projects[workflow.ProjectID]
	if !ok {
		return ErrNotFound
	}
	s.state.Workflows[workflow.ProjectID] = clone(workflow)
	project.UpdatedAt = workflow.UpdatedAt
	s.state.Projects[project.ID] = project
	return s.persistLocked()
}

func (s *Store) ListProviders(ctx context.Context) ([]domain.Provider, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Provider, 0, len(s.state.Providers))
	for _, item := range s.state.Providers {
		items = append(items, clone(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) SaveProvider(ctx context.Context, provider domain.Provider) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Providers[provider.ID] = clone(provider)
	return s.persistLocked()
}

func (s *Store) Provider(ctx context.Context, id string) (domain.Provider, error) {
	if err := ctx.Err(); err != nil {
		return domain.Provider{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.state.Providers[id]
	if !ok {
		return domain.Provider{}, ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.Providers[id]; !ok {
		return ErrNotFound
	}
	delete(s.state.Providers, id)
	return s.persistLocked()
}

func (s *Store) SaveRun(ctx context.Context, run domain.Run) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Runs[run.ID] = clone(run)
	return s.persistLocked()
}

func (s *Store) Run(ctx context.Context, id string) (domain.Run, error) {
	if err := ctx.Err(); err != nil {
		return domain.Run{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.state.Runs[id]
	if !ok {
		return domain.Run{}, ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) UpdateRun(ctx context.Context, id string, update func(*domain.Run)) (domain.Run, error) {
	if err := ctx.Err(); err != nil {
		return domain.Run{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.state.Runs[id]
	if !ok {
		return domain.Run{}, ErrNotFound
	}
	update(&run)
	s.state.Runs[id] = clone(run)
	if err := s.persistLocked(); err != nil {
		return domain.Run{}, err
	}
	return clone(run), nil
}

func (s *Store) MarkActiveRunsInterrupted(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	changed := false
	for id, run := range s.state.Runs {
		if run.Status != domain.RunQueued && run.Status != domain.RunRunning {
			continue
		}
		run.Status = domain.RunInterrupted
		run.Message = "服务重启，任务已中断"
		run.InterruptedAt = &now
		run.FinishedAt = &now
		s.state.Runs[id] = clone(run)
		changed = true
	}
	if !changed {
		return nil
	}
	return s.persistLocked()
}

func (s *Store) Close() error { return nil }
