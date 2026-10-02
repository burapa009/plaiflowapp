ALTER TABLE organizations
    ADD COLUMN branch_code text NOT NULL DEFAULT '',
    ADD COLUMN branch_name text NOT NULL DEFAULT '';
