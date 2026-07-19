CREATE TABLE IF NOT EXISTS projects (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX idx_projects_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS workflows (
  project_id VARCHAR(64) PRIMARY KEY,
  id VARCHAR(64) NOT NULL,
  revision INT NOT NULL,
  document JSON NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_workflows_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS providers (
  id VARCHAR(64) PRIMARY KEY,
  document JSON NOT NULL,
  secret_ciphertext TEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX idx_providers_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS runs (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  document JSON NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX idx_runs_project_id (project_id),
  INDEX idx_runs_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS run_nodes (
  run_id VARCHAR(64) NOT NULL,
  node_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  document JSON NOT NULL,
  PRIMARY KEY (run_id, node_id),
  CONSTRAINT fk_run_nodes_run FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
  INDEX idx_run_nodes_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS assets (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  run_id VARCHAR(64) NOT NULL,
  kind VARCHAR(32) NOT NULL,
  path TEXT NOT NULL,
  document JSON NOT NULL,
  created_at DATETIME(6) NOT NULL,
  INDEX idx_assets_project_id (project_id),
  INDEX idx_assets_run_id (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
