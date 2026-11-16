CREATE TABLE periods (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  code            CHAR(7) NOT NULL,
  status          ENUM('open','closed') NOT NULL DEFAULT 'open',
  closed_by       BIGINT UNSIGNED NULL,
  closed_at       DATETIME NULL,
  closing_balance BIGINT NULL,
  UNIQUE KEY uq_periods_code (code)
) ENGINE=InnoDB;

CREATE TABLE voucher_counters (
  period_id BIGINT UNSIGNED PRIMARY KEY,
  next_no   BIGINT UNSIGNED NOT NULL DEFAULT 1
) ENGINE=InnoDB;

CREATE TABLE funds (
  id       BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name     VARCHAR(120) NOT NULL,
  balance  BIGINT NOT NULL DEFAULT 0,
  archived TINYINT(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE labels (
  id       BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name     VARCHAR(80) NOT NULL,
  archived TINYINT(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE entries (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  period_id  BIGINT UNSIGNED NOT NULL,
  fund_id    BIGINT UNSIGNED NOT NULL,
  voucher_no BIGINT UNSIGNED NOT NULL,
  entry_date DATE NOT NULL,
  kind       ENUM('receipt','payment') NOT NULL,
  amount     BIGINT NOT NULL,
  memo       VARCHAR(255) NOT NULL DEFAULT '',
  source     VARCHAR(16) NOT NULL DEFAULT 'manual',
  created_by BIGINT UNSIGNED NOT NULL,
  deleted_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_entries_voucher (period_id, voucher_no),
  KEY idx_entries_period_kind (period_id, kind)
) ENGINE=InnoDB;

CREATE TABLE entry_labels (
  entry_id BIGINT UNSIGNED NOT NULL,
  label_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (entry_id, label_id)
) ENGINE=InnoDB;

CREATE TABLE expense_requests (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  period_id     BIGINT UNSIGNED NOT NULL,
  fund_id       BIGINT UNSIGNED NOT NULL,
  department_id BIGINT UNSIGNED NOT NULL,
  requester_id  BIGINT UNSIGNED NOT NULL,
  approver_id   BIGINT UNSIGNED NOT NULL,
  amount        BIGINT NOT NULL,
  reason        VARCHAR(255) NOT NULL,
  status        ENUM('pending','approved','rejected','cancelled','paid') NOT NULL DEFAULT 'pending',
  decided_at    DATETIME NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_requests_period (period_id, status),
  KEY idx_requests_fund (fund_id, status),
  KEY idx_requests_department (department_id, period_id)
) ENGINE=InnoDB;

CREATE TABLE bank_events (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  provider_ref VARCHAR(64) NOT NULL,
  fund_id      BIGINT UNSIGNED NOT NULL,
  amount       BIGINT NOT NULL,
  received_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_bank_events_ref (provider_ref)
) ENGINE=InnoDB;

CREATE TABLE payout_receipts (
  id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  provider           VARCHAR(32) NOT NULL,
  event_id           VARCHAR(64) NOT NULL,
  expense_request_id BIGINT UNSIGNED NOT NULL,
  received_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_payout_receipts_event (provider, event_id)
) ENGINE=InnoDB;
