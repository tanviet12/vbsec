CREATE TABLE companies (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(190) NOT NULL,
  billing_email VARCHAR(190) NOT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  email         VARCHAR(190) NOT NULL UNIQUE,
  name          VARCHAR(190) NOT NULL,
  password_hash VARCHAR(255) NOT NULL
);

CREATE TABLE members (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id   BIGINT NOT NULL,
  user_id      BIGINT NOT NULL,
  role         ENUM('owner','admin','accountant','member','client') NOT NULL DEFAULT 'member',
  status       ENUM('invited','active','left') NOT NULL DEFAULT 'invited',
  invite_token CHAR(48) NULL,
  invited_at   DATETIME NULL,
  UNIQUE KEY uq_member (company_id, user_id),
  KEY idx_invite (invite_token)
);

CREATE TABLE member_permissions (
  company_id BIGINT NOT NULL,
  member_id  BIGINT NOT NULL,
  permission VARCHAR(40) NOT NULL,
  granted_by BIGINT NOT NULL,
  granted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (member_id, permission),
  KEY idx_company (company_id)
);

CREATE TABLE sessions (
  token_hash CHAR(64) PRIMARY KEY,
  member_id  BIGINT NOT NULL,
  kind       ENUM('web','assistant') NOT NULL DEFAULT 'web',
  expires_at DATETIME NOT NULL
);

CREATE TABLE projects (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id        BIGINT NOT NULL,
  code              VARCHAR(20) NOT NULL,
  name              VARCHAR(190) NOT NULL,
  manager_id        BIGINT NOT NULL,
  budget_cents      BIGINT NOT NULL DEFAULT 0,
  hourly_rate_cents BIGINT NOT NULL DEFAULT 0,
  notes             TEXT NULL,
  archived_at       DATETIME NULL,
  UNIQUE KEY uq_project_code (company_id, code)
);

CREATE TABLE project_members (
  company_id BIGINT NOT NULL,
  project_id BIGINT NOT NULL,
  member_id  BIGINT NOT NULL,
  PRIMARY KEY (project_id, member_id)
);

CREATE TABLE tasks (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id  BIGINT NOT NULL,
  project_id  BIGINT NOT NULL,
  title       VARCHAR(255) NOT NULL,
  status      ENUM('todo','doing','done') NOT NULL DEFAULT 'todo',
  assignee_id BIGINT NULL,
  KEY idx_project (company_id, project_id)
);

CREATE TABLE comments (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  task_id    BIGINT NOT NULL,
  author_id  BIGINT NOT NULL,
  body       TEXT NOT NULL,
  visibility ENUM('team','client') NOT NULL DEFAULT 'team',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_task (company_id, task_id)
);

CREATE TABLE invoices (
  id             BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id     BIGINT NOT NULL,
  project_code   VARCHAR(20) NOT NULL,
  number         VARCHAR(30) NOT NULL,
  customer_name  VARCHAR(190) NOT NULL,
  customer_email VARCHAR(190) NOT NULL,
  customer_note  VARCHAR(500) NULL,
  pay_token      CHAR(48) NOT NULL,
  amount_cents   BIGINT NOT NULL,
  status         ENUM('draft','sent','paid','void') NOT NULL DEFAULT 'draft',
  prev_status    VARCHAR(10) NULL,
  due_date       DATE NOT NULL,
  issued_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  voided_at      DATETIME NULL,
  UNIQUE KEY uq_invoice_number (company_id, number),
  UNIQUE KEY uq_pay_token (pay_token)
);

CREATE TABLE expenses (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id   BIGINT NOT NULL,
  project_id   BIGINT NOT NULL,
  member_id    BIGINT NOT NULL,
  amount_cents BIGINT NOT NULL,
  note         VARCHAR(255) NOT NULL DEFAULT '',
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE export_jobs (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id   BIGINT NOT NULL,
  requested_by BIGINT NOT NULL,
  kind         VARCHAR(20) NOT NULL,
  date_from    DATE NULL,
  date_to      DATE NULL,
  status       ENUM('pending','done','failed') NOT NULL DEFAULT 'pending',
  file_path    VARCHAR(255) NULL,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE action_confirmations (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  company_id BIGINT NOT NULL,
  member_id  BIGINT NOT NULL,
  action     VARCHAR(40) NOT NULL,
  target     BIGINT NOT NULL,
  code_hash  CHAR(64) NOT NULL,
  attempts   INT NOT NULL DEFAULT 0,
  expires_at DATETIME NOT NULL,
  used_at    DATETIME NULL
);
