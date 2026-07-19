package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"

	"manga-drama-studio/internal/config"
	"manga-drama-studio/internal/domain"
)

type MySQLStore struct {
	db *sql.DB
}

func OpenMySQL(ctx context.Context, database config.Database) (Repository, error) {
	driverConfig := mysqlDriver.NewConfig()
	driverConfig.User = database.User
	driverConfig.Passwd = database.Password
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(database.Host, database.Port)
	driverConfig.DBName = database.Name
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	driverConfig.Params = map[string]string{}
	if database.Params != "" {
		params, err := url.ParseQuery(database.Params)
		if err != nil {
			return nil, fmt.Errorf("parse DATABASE_PARAMS: %w", err)
		}
		for key, values := range params {
			if len(values) == 0 || key == "parseTime" || key == "loc" {
				continue
			}
			driverConfig.Params[key] = values[len(values)-1]
		}
	}

	db, err := sql.Open("mysql", driverConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect mysql: %w", err)
	}
	if err := migrateMySQL(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) CreateProject(ctx context.Context, project domain.Project, workflow domain.Workflow) error {
	document, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("encode workflow: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO projects (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		project.ID, project.Name, project.Description, project.CreatedAt, project.UpdatedAt); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflows (project_id, id, revision, document, updated_at) VALUES (?, ?, ?, ?, ?)`,
		workflow.ProjectID, workflow.ID, workflow.Revision, document, workflow.UpdatedAt); err != nil {
		return fmt.Errorf("create workflow: %w", err)
	}
	return tx.Commit()
}

func (s *MySQLStore) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at FROM projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := []domain.Project{}
	for rows.Next() {
		var project domain.Project
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func (s *MySQLStore) Project(ctx context.Context, id string) (domain.Project, error) {
	var project domain.Project
	err := s.db.QueryRowContext(ctx, `SELECT id, name, description, created_at, updated_at FROM projects WHERE id = ?`, id).
		Scan(&project.ID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt)
	return project, repositoryError(err)
}

func (s *MySQLStore) Workflow(ctx context.Context, projectID string) (domain.Workflow, error) {
	var document []byte
	err := s.db.QueryRowContext(ctx, `SELECT document FROM workflows WHERE project_id = ?`, projectID).Scan(&document)
	if err != nil {
		return domain.Workflow{}, repositoryError(err)
	}
	var workflow domain.Workflow
	if err := json.Unmarshal(document, &workflow); err != nil {
		return domain.Workflow{}, fmt.Errorf("decode workflow: %w", err)
	}
	return workflow, nil
}

func (s *MySQLStore) SaveWorkflow(ctx context.Context, workflow domain.Workflow) error {
	document, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("encode workflow: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE projects SET updated_at = ? WHERE id = ?`, workflow.UpdatedAt, workflow.ProjectID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO workflows (project_id, id, revision, document, updated_at) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = VALUES(id), revision = VALUES(revision), document = VALUES(document), updated_at = VALUES(updated_at)`,
		workflow.ProjectID, workflow.ID, workflow.Revision, document, workflow.UpdatedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) ListProviders(ctx context.Context) ([]domain.Provider, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT document, secret_ciphertext FROM providers ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	providers := []domain.Provider{}
	for rows.Next() {
		provider, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (s *MySQLStore) SaveProvider(ctx context.Context, provider domain.Provider) error {
	document, err := json.Marshal(provider)
	if err != nil {
		return fmt.Errorf("encode provider: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO providers (id, document, secret_ciphertext, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE document = VALUES(document), secret_ciphertext = VALUES(secret_ciphertext), updated_at = VALUES(updated_at)`,
		provider.ID, document, provider.SecretCiphertext, provider.CreatedAt, provider.UpdatedAt)
	return err
}

func (s *MySQLStore) Provider(ctx context.Context, id string) (domain.Provider, error) {
	row := s.db.QueryRowContext(ctx, `SELECT document, secret_ciphertext FROM providers WHERE id = ?`, id)
	provider, err := scanProvider(row)
	return provider, repositoryError(err)
}

func (s *MySQLStore) DeleteProvider(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MySQLStore) SaveRun(ctx context.Context, run domain.Run) error {
	document, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("encode run: %w", err)
	}
	updatedAt := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `INSERT INTO runs (id, project_id, status, document, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE project_id = VALUES(project_id), status = VALUES(status), document = VALUES(document), updated_at = VALUES(updated_at)`,
		run.ID, run.ProjectID, run.Status, document, run.CreatedAt, updatedAt)
	return err
}

func (s *MySQLStore) Run(ctx context.Context, id string) (domain.Run, error) {
	var document []byte
	err := s.db.QueryRowContext(ctx, `SELECT document FROM runs WHERE id = ?`, id).Scan(&document)
	if err != nil {
		return domain.Run{}, repositoryError(err)
	}
	return decodeRun(document)
}

func (s *MySQLStore) UpdateRun(ctx context.Context, id string, update func(*domain.Run)) (domain.Run, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Run{}, err
	}
	defer tx.Rollback()
	var document []byte
	if err := tx.QueryRowContext(ctx, `SELECT document FROM runs WHERE id = ? FOR UPDATE`, id).Scan(&document); err != nil {
		return domain.Run{}, repositoryError(err)
	}
	run, err := decodeRun(document)
	if err != nil {
		return domain.Run{}, err
	}
	update(&run)
	document, err = json.Marshal(run)
	if err != nil {
		return domain.Run{}, fmt.Errorf("encode run: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE runs SET project_id = ?, status = ?, document = ?, updated_at = ? WHERE id = ?`,
		run.ProjectID, run.Status, document, time.Now().UTC(), run.ID); err != nil {
		return domain.Run{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Run{}, err
	}
	return run, nil
}

func (s *MySQLStore) MarkActiveRunsInterrupted(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id, document FROM runs WHERE status IN (?, ?) FOR UPDATE`, domain.RunQueued, domain.RunRunning)
	if err != nil {
		return err
	}
	type activeRun struct {
		id       string
		document []byte
	}
	active := []activeRun{}
	for rows.Next() {
		var item activeRun
		if err := rows.Scan(&item.id, &item.document); err != nil {
			rows.Close()
			return err
		}
		active = append(active, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	sort.Slice(active, func(i, j int) bool { return active[i].id < active[j].id })
	now := time.Now().UTC()
	for _, item := range active {
		run, err := decodeRun(item.document)
		if err != nil {
			return err
		}
		run.Status = domain.RunInterrupted
		run.Message = "服务重启，任务已中断"
		run.InterruptedAt = &now
		run.FinishedAt = &now
		document, err := json.Marshal(run)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE runs SET status = ?, document = ?, updated_at = ? WHERE id = ?`, run.Status, document, now, item.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) SaveAsset(ctx context.Context, asset domain.Asset) error {
	document, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("encode asset: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO assets (id, project_id, run_id, kind, path, document, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE kind = VALUES(kind), path = VALUES(path), document = VALUES(document)`,
		asset.ID, asset.ProjectID, asset.RunID, asset.Kind, asset.Path, document, asset.CreatedAt)
	return err
}

func (s *MySQLStore) Asset(ctx context.Context, id string) (domain.Asset, error) {
	var document []byte
	var path string
	err := s.db.QueryRowContext(ctx, `SELECT document, path FROM assets WHERE id = ?`, id).Scan(&document, &path)
	if err != nil {
		return domain.Asset{}, repositoryError(err)
	}
	var asset domain.Asset
	if err := json.Unmarshal(document, &asset); err != nil {
		return domain.Asset{}, fmt.Errorf("decode asset: %w", err)
	}
	asset.Path = path
	return asset, nil
}

func (s *MySQLStore) ListRunAssets(ctx context.Context, runID string) ([]domain.Asset, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT document, path FROM assets WHERE run_id = ? ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Asset{}
	for rows.Next() {
		var document []byte
		var path string
		if err := rows.Scan(&document, &path); err != nil {
			return nil, err
		}
		var asset domain.Asset
		if err := json.Unmarshal(document, &asset); err != nil {
			return nil, err
		}
		asset.Path = path
		items = append(items, asset)
	}
	return items, rows.Err()
}

type scanner interface {
	Scan(...any) error
}

func scanProvider(row scanner) (domain.Provider, error) {
	var document []byte
	var secret string
	if err := row.Scan(&document, &secret); err != nil {
		return domain.Provider{}, err
	}
	var provider domain.Provider
	if err := json.Unmarshal(document, &provider); err != nil {
		return domain.Provider{}, fmt.Errorf("decode provider: %w", err)
	}
	provider.SecretCiphertext = secret
	return provider, nil
}

func decodeRun(document []byte) (domain.Run, error) {
	var run domain.Run
	if err := json.Unmarshal(document, &run); err != nil {
		return domain.Run{}, fmt.Errorf("decode run: %w", err)
	}
	return run, nil
}

func repositoryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil && strings.Contains(err.Error(), "context canceled") {
		return context.Canceled
	}
	return err
}
