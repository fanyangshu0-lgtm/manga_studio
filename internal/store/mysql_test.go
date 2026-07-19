package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"manga-drama-studio/internal/config"
	"manga-drama-studio/internal/domain"
)

func TestMySQLRepositoryContract(t *testing.T) {
	database := mysqlTestConfig(t)
	repository, err := OpenMySQL(context.Background(), database)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, repository.Close()) })

	now := time.Now().UTC().Truncate(time.Microsecond)
	project := domain.Project{ID: domain.NewID("prj"), Name: "MySQL 项目", CreatedAt: now, UpdatedAt: now}
	workflow := domain.Workflow{ID: domain.NewID("wf"), ProjectID: project.ID, Revision: 1, UpdatedAt: now}
	require.NoError(t, repository.CreateProject(context.Background(), project, workflow))

	got, err := repository.Project(context.Background(), project.ID)
	require.NoError(t, err)
	assert.Equal(t, project.Name, got.Name)
	graph, err := repository.Workflow(context.Background(), project.ID)
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, graph.ID)
}

func TestMySQLMarksActiveRunsInterrupted(t *testing.T) {
	database := mysqlTestConfig(t)
	repository, err := OpenMySQL(context.Background(), database)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, repository.Close()) })

	run := domain.Run{ID: domain.NewID("run"), Status: domain.RunRunning, CreatedAt: time.Now().UTC()}
	require.NoError(t, repository.SaveRun(context.Background(), run))
	require.NoError(t, repository.MarkActiveRunsInterrupted(context.Background()))

	got, err := repository.Run(context.Background(), run.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RunInterrupted, got.Status)
	assert.NotNil(t, got.InterruptedAt)
}

func mysqlTestConfig(t *testing.T) config.Database {
	t.Helper()
	if os.Getenv("TEST_MYSQL_HOST") == "" {
		t.Skip("TEST_MYSQL_HOST is required")
	}
	return config.Database{
		Host: os.Getenv("TEST_MYSQL_HOST"), Port: "13307", Name: "manga_drama_studio_test",
		User: "manga_test", Password: "manga_test_password", Params: "charset=utf8mb4&parseTime=true&loc=UTC",
	}
}
