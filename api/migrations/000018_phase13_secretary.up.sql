-- Phase 13 company pilot. All public routes remain behind SECRETARY_ENABLED.
ALTER TABLE durable_jobs DROP CONSTRAINT durable_jobs_kind_check;
ALTER TABLE durable_jobs ADD CONSTRAINT durable_jobs_kind_check CHECK (kind IN ('export','ocr','secretary'));

CREATE TABLE secretary_briefings (
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    local_day date NOT NULL,
    job_id uuid NOT NULL UNIQUE REFERENCES durable_jobs(id),
    status text NOT NULL CHECK (status IN ('Queued','Ready','Failed')),
    role text NOT NULL CHECK (role IN ('Owner','Admin','Member')),
    items jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(items)='array'),
    remaining_count integer NOT NULL DEFAULT 0 CHECK (remaining_count>=0),
    requested_at timestamptz NOT NULL,
    generated_at timestamptz,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,user_id,local_day),
    FOREIGN KEY (organization_id,user_id) REFERENCES memberships(organization_id,user_id) ON DELETE CASCADE
);
CREATE INDEX secretary_briefings_expiry_idx ON secretary_briefings(expires_at);
ALTER TABLE secretary_briefings ENABLE ROW LEVEL SECURITY;
CREATE POLICY secretary_briefing_self ON secretary_briefings FOR ALL
    USING (organization_id=app_organization_id() AND user_id=app_user_id()
        AND is_organization_member(app_user_id(),organization_id))
    WITH CHECK (organization_id=app_organization_id() AND user_id=app_user_id()
        AND is_organization_member(app_user_id(),organization_id));

CREATE TABLE secretary_preferences (
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    hidden_categories text[] NOT NULL DEFAULT '{}',
    pinned_task_ids uuid[] NOT NULL DEFAULT '{}',
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,user_id),
    FOREIGN KEY (organization_id,user_id) REFERENCES memberships(organization_id,user_id) ON DELETE CASCADE,
    CHECK (hidden_categories <@ ARRAY['Task','FollowUp','MeetingAction','Routine']::text[])
);
ALTER TABLE secretary_preferences ENABLE ROW LEVEL SECURITY;
CREATE POLICY secretary_preferences_self ON secretary_preferences FOR ALL
    USING (organization_id=app_organization_id() AND user_id=app_user_id()
        AND is_organization_member(app_user_id(),organization_id))
    WITH CHECK (organization_id=app_organization_id() AND user_id=app_user_id()
        AND is_organization_member(app_user_id(),organization_id));

ALTER TABLE tasks ADD COLUMN secretary_category text NOT NULL DEFAULT 'Task'
    CHECK (secretary_category IN ('Task','FollowUp','MeetingAction'));
ALTER TABLE tasks ADD COLUMN follow_up_with text NOT NULL DEFAULT '' CHECK (length(follow_up_with)<=200);
ALTER TABLE tasks ADD COLUMN meeting_name text NOT NULL DEFAULT '' CHECK (length(meeting_name)<=200);
ALTER TABLE tasks ADD COLUMN meeting_on date;
ALTER TABLE tasks ADD CONSTRAINT task_secretary_source_consistent CHECK (
    (secretary_category='FollowUp' AND length(btrim(follow_up_with))>0 AND meeting_name='' AND meeting_on IS NULL)
    OR (secretary_category='MeetingAction' AND length(btrim(meeting_name))>0 AND meeting_on IS NOT NULL AND follow_up_with='')
    OR (secretary_category='Task' AND follow_up_with='' AND meeting_name='' AND meeting_on IS NULL)
);

CREATE TABLE routine_templates (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    title text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 200),
    responsible_user_id uuid NOT NULL,
    cadence text NOT NULL CHECK (cadence IN ('Weekly','Monthly','Quarterly')),
    first_due_on date NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (organization_id,id),
    FOREIGN KEY (organization_id,responsible_user_id) REFERENCES memberships(organization_id,user_id) ON DELETE CASCADE
);
CREATE INDEX routine_templates_responsible_idx ON routine_templates(organization_id,responsible_user_id,active);
ALTER TABLE routine_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY routine_templates_read ON routine_templates FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND (responsible_user_id=app_user_id() OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin')));
CREATE POLICY routine_templates_manage ON routine_templates FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));

CREATE TABLE routine_suggestions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    template_id uuid NOT NULL,
    due_on date NOT NULL,
    status text NOT NULL CHECK (status IN ('Pending','Confirmed','Skipped')),
    task_id uuid,
    created_at timestamptz NOT NULL,
    resolved_at timestamptz,
    UNIQUE (organization_id,template_id,due_on),
    FOREIGN KEY (organization_id,template_id) REFERENCES routine_templates(organization_id,id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id,task_id) REFERENCES tasks(organization_id,id)
);
CREATE INDEX routine_suggestions_latest_idx ON routine_suggestions(organization_id,template_id,due_on DESC);
ALTER TABLE routine_suggestions ENABLE ROW LEVEL SECURITY;
CREATE POLICY routine_suggestions_read ON routine_suggestions FOR SELECT USING (
    organization_id=app_organization_id() AND EXISTS (
        SELECT 1 FROM routine_templates t WHERE t.organization_id=routine_suggestions.organization_id
        AND t.id=routine_suggestions.template_id AND
        (t.responsible_user_id=app_user_id() OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))));
CREATE POLICY routine_suggestions_write ON routine_suggestions FOR ALL USING (
    organization_id=app_organization_id() AND EXISTS (
        SELECT 1 FROM routine_templates t WHERE t.organization_id=routine_suggestions.organization_id
        AND t.id=routine_suggestions.template_id AND
        (t.responsible_user_id=app_user_id() OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))))
    WITH CHECK (organization_id=app_organization_id() AND EXISTS (
        SELECT 1 FROM routine_templates t WHERE t.organization_id=routine_suggestions.organization_id
        AND t.id=routine_suggestions.template_id AND
        (t.responsible_user_id=app_user_id() OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))));
