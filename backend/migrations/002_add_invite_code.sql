ALTER TABLE kitchen_invites ADD COLUMN code TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_kitchen_invites_code ON kitchen_invites(code);
