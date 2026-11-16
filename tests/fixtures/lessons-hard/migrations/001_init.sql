SET sql_mode = 'STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION';

CREATE TABLE companies (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(160) NOT NULL,
  status ENUM('active','suspended') NOT NULL DEFAULT 'active'
);

CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(190) NOT NULL UNIQUE,
  display_name VARCHAR(160) NOT NULL,
  password_hash VARCHAR(100) NOT NULL,
  phone VARCHAR(32) NULL,
  bank_account VARCHAR(64) NULL,
  active TINYINT NOT NULL DEFAULT 1
);

CREATE TABLE members (
  company_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  role ENUM('owner','admin','member') NOT NULL,
  status ENUM('active','left') NOT NULL DEFAULT 'active',
  PRIMARY KEY (company_id, user_id)
);

CREATE TABLE member_permissions (
  company_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  permission VARCHAR(40) NOT NULL,
  PRIMARY KEY (company_id, user_id, permission)
);

CREATE TABLE two_factor (
  user_id BIGINT PRIMARY KEY,
  secret_enc VARBINARY(128) NOT NULL,
  enabled_at DATETIME NOT NULL
);

CREATE TABLE sessions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  company_id BIGINT NOT NULL,
  token_hash CHAR(64) NOT NULL UNIQUE,
  expires_at DATETIME NOT NULL
);

CREATE TABLE password_resets (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  token VARCHAR(64) NOT NULL UNIQUE,
  expires_at DATETIME NOT NULL,
  used_at DATETIME NULL
);

CREATE TABLE invites (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  email VARCHAR(190) NOT NULL,
  role ENUM('admin','member') NOT NULL,
  token_hash CHAR(64) NOT NULL UNIQUE,
  invited_by BIGINT NOT NULL,
  expires_at DATETIME NOT NULL,
  accepted_at DATETIME NULL
);

CREATE TABLE rate_hits (
  bucket VARCHAR(255) NOT NULL,
  hit_at DATETIME NOT NULL,
  KEY (bucket, hit_at)
);

CREATE TABLE security_events (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email_hash CHAR(64) NOT NULL,
  kind VARCHAR(32) NOT NULL,
  ip VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL,
  KEY (email_hash, created_at)
);

CREATE TABLE projects (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  name VARCHAR(160) NOT NULL,
  status ENUM('open','closed') NOT NULL DEFAULT 'open',
  budget_cents BIGINT NOT NULL DEFAULT 0,
  closing_summary VARCHAR(4000) NULL
);

CREATE TABLE project_notes (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  project_id BIGINT NOT NULL,
  author_id BIGINT NOT NULL,
  body VARCHAR(500) NOT NULL
);

CREATE TABLE files (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  uploader_id BIGINT NOT NULL,
  original_name VARCHAR(255) NOT NULL,
  sha256 CHAR(64) NOT NULL,
  storage_key CHAR(32) NOT NULL,
  status ENUM('temp','attached') NOT NULL DEFAULT 'temp',
  created_at DATETIME NOT NULL
);

CREATE TABLE requests (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  requester_id BIGINT NOT NULL,
  approver_id BIGINT NULL,
  amount_cents BIGINT NOT NULL,
  status ENUM('draft','pending','approved','rejected') NOT NULL DEFAULT 'draft',
  receipt_file_id BIGINT NULL
);

CREATE TABLE categories (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  name VARCHAR(80) NOT NULL,
  UNIQUE KEY (company_id, name)
);

CREATE TABLE expenses (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  category_id BIGINT NOT NULL,
  spent_on DATE NOT NULL,
  amount_cents BIGINT NOT NULL,
  created_by BIGINT NOT NULL
);

CREATE TABLE company_settings (
  company_id BIGINT PRIMARY KEY,
  approval_mode ENUM('strict','relaxed') NOT NULL DEFAULT 'strict',
  monthly_limit_cents BIGINT NOT NULL DEFAULT 0,
  retention_days INT NOT NULL DEFAULT 365
);

CREATE TABLE setting_history (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  name VARCHAR(40) NOT NULL,
  old_value VARCHAR(64) NOT NULL,
  new_value VARCHAR(64) NOT NULL,
  changed_by BIGINT NOT NULL,
  changed_at DATETIME NOT NULL
);

CREATE TABLE activity (
  id BIGINT NOT NULL AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  actor_id BIGINT NOT NULL,
  action VARCHAR(40) NOT NULL,
  details VARCHAR(1000) NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id, created_at),
  KEY (company_id, created_at)
)
PARTITION BY RANGE (TO_DAYS(created_at)) (
  PARTITION p2024 VALUES LESS THAN (TO_DAYS('2025-01-01')),
  PARTITION p2025 VALUES LESS THAN (TO_DAYS('2026-01-01')),
  PARTITION pmax VALUES LESS THAN MAXVALUE
);

CREATE TABLE ops_access_log (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  operator_id BIGINT NOT NULL,
  company_id BIGINT NOT NULL,
  kind VARCHAR(40) NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE agent_connections (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  client_id VARCHAR(64) NOT NULL,
  token_hash CHAR(64) NOT NULL UNIQUE,
  scopes VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL,
  revoked_at DATETIME NULL
);

CREATE TABLE special_grants (
  company_id BIGINT NOT NULL,
  client_id VARCHAR(64) NOT NULL,
  user_id BIGINT NOT NULL,
  grant_name VARCHAR(40) NOT NULL,
  approved_by BIGINT NOT NULL,
  approved_at DATETIME NOT NULL,
  PRIMARY KEY (company_id, client_id, user_id, grant_name)
);

CREATE TABLE drafts (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  author_id BIGINT NOT NULL,
  title VARCHAR(160) NOT NULL,
  body TEXT NOT NULL,
  status ENUM('pending','published') NOT NULL DEFAULT 'pending'
);

CREATE TABLE draft_codes (
  draft_id BIGINT NOT NULL,
  connection_id BIGINT NOT NULL,
  code_hash CHAR(64) NOT NULL,
  expires_at DATETIME NOT NULL,
  PRIMARY KEY (draft_id, connection_id)
);

CREATE TABLE payroll_lines (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  period CHAR(7) NOT NULL,
  net_cents BIGINT NOT NULL
);
