CREATE TABLE users (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  email         VARCHAR(190) NOT NULL,
  password_hash VARCHAR(100) NOT NULL,
  role          ENUM('admin','accountant','manager','staff') NOT NULL DEFAULT 'staff',
  totp_secret   VARBINARY(128) NOT NULL,
  active        TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB;

CREATE TABLE sessions (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  token_hash CHAR(64) NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_sessions_token (token_hash),
  KEY idx_sessions_user (user_id)
) ENGINE=InnoDB;

CREATE TABLE api_tokens (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id      BIGINT UNSIGNED NOT NULL,
  name         VARCHAR(80) NOT NULL,
  token_hash   CHAR(64) NOT NULL,
  last_used_at DATETIME NULL,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_api_tokens_hash (token_hash),
  KEY idx_api_tokens_user (user_id)
) ENGINE=InnoDB;

CREATE TABLE otp_attempts (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  ok         TINYINT(1) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_otp_attempts_user (user_id, created_at)
) ENGINE=InnoDB;

CREATE TABLE departments (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(120) NOT NULL,
  head_id       BIGINT UNSIGNED NOT NULL,
  monthly_limit BIGINT NOT NULL DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE employees (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id       BIGINT UNSIGNED NOT NULL,
  department_id BIGINT UNSIGNED NOT NULL,
  full_name     VARCHAR(160) NOT NULL,
  tax_code      VARBINARY(255) NULL,
  bank_account  VARBINARY(255) NULL,
  base_salary   BIGINT NOT NULL DEFAULT 0,
  status        ENUM('active','terminated') NOT NULL DEFAULT 'active',
  terminated_at DATETIME NULL,
  UNIQUE KEY uq_employees_user (user_id),
  KEY idx_employees_department (department_id)
) ENGINE=InnoDB;

CREATE TABLE invites (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  email         VARCHAR(190) NOT NULL,
  token_hash    CHAR(64) NOT NULL,
  department_id BIGINT UNSIGNED NOT NULL,
  created_by    BIGINT UNSIGNED NOT NULL,
  expires_at    DATETIME NOT NULL,
  used_at       DATETIME NULL,
  UNIQUE KEY uq_invites_token (token_hash)
) ENGINE=InnoDB;

CREATE TABLE audit_log (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  actor_id   BIGINT UNSIGNED NOT NULL,
  action     VARCHAR(64) NOT NULL,
  target_id  BIGINT UNSIGNED NOT NULL,
  detail     JSON NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_audit_action (action, created_at)
) ENGINE=InnoDB;
