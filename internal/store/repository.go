package store

import (
	"context"

	"manga-drama-studio/internal/domain"
)

type Repository interface {
	CreateProject(context.Context, domain.Project, domain.Workflow) error
	ListProjects(context.Context) ([]domain.Project, error)
	Project(context.Context, string) (domain.Project, error)
	Workflow(context.Context, string) (domain.Workflow, error)
	SaveWorkflow(context.Context, domain.Workflow) error
	ListProviders(context.Context) ([]domain.Provider, error)
	SaveProvider(context.Context, domain.Provider) error
	Provider(context.Context, string) (domain.Provider, error)
	DeleteProvider(context.Context, string) error
	SaveRun(context.Context, domain.Run) error
	Run(context.Context, string) (domain.Run, error)
	UpdateRun(context.Context, string, func(*domain.Run)) (domain.Run, error)
	SaveAsset(context.Context, domain.Asset) error
	Asset(context.Context, string) (domain.Asset, error)
	ListRunAssets(context.Context, string) ([]domain.Asset, error)
	MarkActiveRunsInterrupted(context.Context) error
	Close() error
}
