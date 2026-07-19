package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"manga-drama-studio/internal/domain"
)

func TestJSONStoreImplementsRepositoryContract(t *testing.T) {
	var _ Repository = (*Store)(nil)

	repository, err := Open(filepath.Join(t.TempDir(), "studio.json"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, repository.Close()) })

	now := time.Now().UTC()
	project := domain.Project{ID: "prj-1", Name: "测试项目", CreatedAt: now, UpdatedAt: now}
	workflow := domain.Workflow{ID: "wf-1", ProjectID: project.ID, UpdatedAt: now}
	require.NoError(t, repository.CreateProject(context.Background(), project, workflow))

	projects, err := repository.ListProjects(context.Background())
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, project.ID, projects[0].ID)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = repository.Project(cancelled, project.ID)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMarkActiveRunsInterrupted(t *testing.T) {
	repository, err := Open(filepath.Join(t.TempDir(), "studio.json"))
	require.NoError(t, err)

	for _, run := range []domain.Run{
		{ID: "queued", Status: domain.RunQueued},
		{ID: "running", Status: domain.RunRunning},
		{ID: "done", Status: domain.RunSucceeded},
	} {
		require.NoError(t, repository.SaveRun(context.Background(), run))
	}

	require.NoError(t, repository.MarkActiveRunsInterrupted(context.Background()))

	for _, id := range []string{"queued", "running"} {
		run, runErr := repository.Run(context.Background(), id)
		require.NoError(t, runErr)
		assert.Equal(t, domain.RunInterrupted, run.Status)
		assert.NotNil(t, run.InterruptedAt)
	}
	done, err := repository.Run(context.Background(), "done")
	require.NoError(t, err)
	assert.Equal(t, domain.RunSucceeded, done.Status)
	assert.Nil(t, done.InterruptedAt)
}

func TestRepositoryReturnsNotFound(t *testing.T) {
	repository, err := Open(filepath.Join(t.TempDir(), "studio.json"))
	require.NoError(t, err)

	_, err = repository.Project(context.Background(), "missing")
	assert.True(t, errors.Is(err, ErrNotFound))
}
