ALTER TABLE organizations
    ADD COLUMN timezone text NOT NULL DEFAULT 'Asia/Bangkok'
        CHECK (length(timezone) BETWEEN 1 AND 64);

CREATE TABLE tasks (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    title text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 200),
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 5000),
    creator_user_id uuid NOT NULL REFERENCES users(id),
    assignee_user_id uuid,
    status text NOT NULL DEFAULT 'Open' CHECK (status IN ('Open','InProgress','Done','Cancelled')),
    priority text NOT NULL DEFAULT 'Normal' CHECK (priority IN ('Normal','High','Urgent')),
    due_on date,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    status_changed_at timestamptz NOT NULL,
    completed_at timestamptz,
    completed_by_user_id uuid REFERENCES users(id),
    UNIQUE (organization_id,id),
    FOREIGN KEY (organization_id,assignee_user_id) REFERENCES memberships(organization_id,user_id),
    CHECK ((status='Done' AND completed_at IS NOT NULL AND completed_by_user_id IS NOT NULL)
        OR (status<>'Done' AND completed_at IS NULL AND completed_by_user_id IS NULL))
);

CREATE TABLE task_watchers (
    organization_id uuid NOT NULL,
    task_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,task_id,user_id),
    FOREIGN KEY (organization_id,task_id) REFERENCES tasks(organization_id,id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id,user_id) REFERENCES memberships(organization_id,user_id)
);

CREATE TABLE domain_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    event_type text NOT NULL CHECK (event_type IN (
        'task.created','task.details_changed','task.assigned','task.unassigned','task.watcher_added',
        'task.watcher_removed','task.due_changed','task.priority_changed','task.status_changed')),
    aggregate_type text NOT NULL DEFAULT 'Task' CHECK (aggregate_type='Task'),
    aggregate_id uuid NOT NULL,
    actor_user_id uuid NOT NULL REFERENCES users(id),
    causation_type text,
    causation_id text,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload)='object'),
    occurred_at timestamptz NOT NULL,
    processing_state text NOT NULL DEFAULT 'Pending' CHECK (processing_state IN ('Pending','Processing','Processed','Failed')),
    available_at timestamptz NOT NULL,
    lease_expires_at timestamptz,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count>=0),
    processed_at timestamptz,
    failure_code text,
    FOREIGN KEY (organization_id,aggregate_id) REFERENCES tasks(organization_id,id)
);

CREATE TABLE reminders (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    task_id uuid NOT NULL,
    recipient_user_id uuid NOT NULL,
    due_on date NOT NULL,
    milestone text NOT NULL CHECK (milestone IN ('day_before','due_day','day_after')),
    scheduled_for timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id,task_id,recipient_user_id,due_on,milestone),
    FOREIGN KEY (organization_id,task_id) REFERENCES tasks(organization_id,id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id,recipient_user_id) REFERENCES memberships(organization_id,user_id)
);

CREATE TABLE notification_preferences (
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    category text NOT NULL CHECK (category IN ('Assignment','TaskChange','DueReminder')),
    web_mode text NOT NULL CHECK (web_mode IN ('Immediate','Digest','Off')),
    line_mode text NOT NULL CHECK (line_mode IN ('Immediate','Digest','Off')),
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,user_id,category),
    FOREIGN KEY (organization_id,user_id) REFERENCES memberships(organization_id,user_id) ON DELETE CASCADE
);

CREATE TABLE notifications (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    recipient_user_id uuid NOT NULL,
    category text NOT NULL CHECK (category IN ('Assignment','TaskChange','DueReminder')),
    logical_key text NOT NULL,
    title text NOT NULL,
    deep_link text NOT NULL,
    delivery_mode text NOT NULL CHECK (delivery_mode IN ('Immediate','Digest','Off')),
    created_at timestamptz NOT NULL,
    read_at timestamptz,
    digested_at timestamptz,
    UNIQUE (organization_id,recipient_user_id,logical_key),
    UNIQUE (organization_id,id),
    FOREIGN KEY (organization_id,recipient_user_id) REFERENCES memberships(organization_id,user_id) ON DELETE CASCADE
);

CREATE TABLE notification_deliveries (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    notification_id uuid NOT NULL,
    channel text NOT NULL CHECK (channel IN ('LINE')),
    logical_key text NOT NULL,
    status text NOT NULL DEFAULT 'Pending' CHECK (status IN ('Pending','Processing','Delivered','Retryable','Failed')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count>=0),
    available_at timestamptz NOT NULL,
    lease_expires_at timestamptz,
    delivered_at timestamptz,
    failure_code text,
    UNIQUE (organization_id,channel,logical_key),
    FOREIGN KEY (organization_id,notification_id) REFERENCES notifications(organization_id,id) ON DELETE CASCADE
);

CREATE TABLE export_jobs (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    requester_user_id uuid NOT NULL REFERENCES users(id),
    filters jsonb NOT NULL CHECK (jsonb_typeof(filters)='object'),
    format_version text NOT NULL DEFAULT 'csv-v1' CHECK (format_version='csv-v1'),
    mode text NOT NULL CHECK (mode IN ('sync','background')),
    status text NOT NULL CHECK (status IN ('Queued','Running','Completed','Failed','Expired')),
    requested_at timestamptz NOT NULL,
    started_at timestamptz,
    completed_at timestamptz,
    failed_at timestamptz,
    expired_at timestamptz,
    authorized_row_count bigint NOT NULL CHECK (authorized_row_count BETWEEN 0 AND 100000),
    produced_byte_count bigint CHECK (produced_byte_count>=0),
    data_as_of timestamptz,
    failure_code text,
    object_key text,
    object_expires_at timestamptz,
    UNIQUE (organization_id,id),
    FOREIGN KEY (organization_id,requester_user_id) REFERENCES memberships(organization_id,user_id)
);

CREATE INDEX tasks_organization_active_due_idx ON tasks (organization_id,status,due_on,id)
    WHERE status IN ('Open','InProgress');
CREATE INDEX tasks_organization_assignee_status_due_idx ON tasks (organization_id,assignee_user_id,status,due_on,id);
CREATE INDEX tasks_organization_created_idx ON tasks (organization_id,created_at DESC,id DESC);
CREATE INDEX task_watchers_member_task_idx ON task_watchers (organization_id,user_id,task_id);
CREATE INDEX domain_events_claim_idx ON domain_events (processing_state,available_at,id)
    WHERE processing_state IN ('Pending','Processing');
CREATE INDEX reminders_due_idx ON reminders (scheduled_for,id);
CREATE INDEX notifications_inbox_idx ON notifications (organization_id,recipient_user_id,read_at,created_at DESC,id DESC);
CREATE INDEX notification_deliveries_claim_idx ON notification_deliveries (status,available_at,id)
    WHERE status IN ('Pending','Retryable','Processing');
CREATE INDEX export_jobs_history_idx ON export_jobs (organization_id,requested_at DESC,id DESC);
CREATE INDEX export_jobs_claim_idx ON export_jobs (status,requested_at,id) WHERE status IN ('Queued','Running');

CREATE FUNCTION phase2_membership_cleanup() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE tasks SET assignee_user_id=NULL,updated_at=now()
        WHERE organization_id=OLD.organization_id AND assignee_user_id=OLD.user_id;
    DELETE FROM task_watchers WHERE organization_id=OLD.organization_id AND user_id=OLD.user_id;
    DELETE FROM reminders WHERE organization_id=OLD.organization_id AND recipient_user_id=OLD.user_id;
    RETURN OLD;
END
$$;

CREATE TRIGGER memberships_phase2_cleanup
    BEFORE DELETE ON memberships
    FOR EACH ROW EXECUTE FUNCTION phase2_membership_cleanup();

CREATE FUNCTION prevent_assignee_watcher_overlap() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM tasks WHERE organization_id=NEW.organization_id AND id=NEW.task_id AND assignee_user_id=NEW.user_id) THEN
        RAISE EXCEPTION 'assignee cannot also be watcher';
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER task_watchers_no_assignee
    BEFORE INSERT OR UPDATE ON task_watchers
    FOR EACH ROW EXECUTE FUNCTION prevent_assignee_watcher_overlap();

ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_watchers ENABLE ROW LEVEL SECURITY;
ALTER TABLE domain_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE reminders ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE export_jobs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tasks_member_policy ON tasks
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id))
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY task_watchers_member_policy ON task_watchers
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id))
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY domain_events_insert_policy ON domain_events FOR INSERT
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY notification_preferences_self_policy ON notification_preferences
    USING (organization_id=app_organization_id() AND user_id=app_user_id())
    WITH CHECK (organization_id=app_organization_id() AND user_id=app_user_id());
CREATE POLICY notifications_self_policy ON notifications
    USING (organization_id=app_organization_id() AND recipient_user_id=app_user_id())
    WITH CHECK (organization_id=app_organization_id() AND recipient_user_id=app_user_id());
CREATE POLICY export_jobs_manager_policy ON export_jobs
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
