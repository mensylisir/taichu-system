-- Fix backup_schedules table schema
ALTER TABLE backup_schedules
ADD COLUMN IF NOT EXISTS name VARCHAR(255),
ADD COLUMN IF NOT EXISTS cron_expr VARCHAR(100),
ADD COLUMN IF NOT EXISTS retention_days INT DEFAULT 7,
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS created_by VARCHAR(100) NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Copy data from old columns to new columns if needed
UPDATE backup_schedules SET name = schedule_name WHERE name IS NULL;
UPDATE backup_schedules SET cron_expr = cron_expression WHERE cron_expr IS NULL;
UPDATE backup_schedules SET retention_days = retention_count WHERE retention_days IS NULL;
