-- Additive and dormant until REVIEW_ENABLED=true. Keep protected values outside audit_events.
ALTER TABLE document_extraction_reviews ADD COLUMN draft_revision integer NOT NULL DEFAULT 0 CHECK (draft_revision >= 0);
CREATE TABLE document_review_drafts (
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    ocr_job_id uuid NOT NULL,
    revision integer NOT NULL CHECK (revision > 0),
    values jsonb NOT NULL CHECK (jsonb_typeof(values) = 'object'),
    decisions jsonb NOT NULL CHECK (jsonb_typeof(decisions) = 'object'),
    updated_by uuid NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,document_id),
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,updated_by) REFERENCES memberships(organization_id,user_id)
);
CREATE TABLE document_review_draft_history (
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    revision integer NOT NULL CHECK (revision > 0),
    ocr_job_id uuid NOT NULL,
    values jsonb NOT NULL CHECK (jsonb_typeof(values) = 'object'),
    decisions jsonb NOT NULL CHECK (jsonb_typeof(decisions) = 'object'),
    updated_by uuid NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,document_id,revision),
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id)
);
CREATE TABLE document_review_tasks (
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    task_id uuid NOT NULL,
    PRIMARY KEY (organization_id,document_id),
    UNIQUE (organization_id,task_id),
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,task_id) REFERENCES tasks(organization_id,id)
);
CREATE TABLE document_review_returns (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    ocr_job_id uuid NOT NULL,
    action text NOT NULL DEFAULT 'Return' CHECK (action IN ('Return','Reprocess')),
    reason_code text NOT NULL CHECK (reason_code IN ('missing_value','incorrect_value','unreadable_original','quality_issue','missing_page','other')),
    private_note text NOT NULL DEFAULT '' CHECK (length(private_note) <= 1000),
    returned_by uuid NOT NULL REFERENCES users(id),
    returned_at timestamptz NOT NULL,
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id)
);
CREATE INDEX document_review_returns_history ON document_review_returns (organization_id,document_id,returned_at DESC,id DESC);
CREATE TABLE document_export_snapshots (
    job_id uuid PRIMARY KEY REFERENCES durable_jobs(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    requester_user_id uuid NOT NULL REFERENCES users(id),
    product text NOT NULL CHECK (product IN ('raw_documents','confirmed_values','approved_suggestions')),
    document_status text NOT NULL CHECK (document_status IN ('Available','Archived','Trash')),
    format text NOT NULL CHECK (format IN ('csv','xlsx')),
    date_from date,
    date_to date,
    row_count integer NOT NULL CHECK (row_count >= 0),
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id,job_id)
);
CREATE TABLE document_export_snapshot_rows (
    job_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal > 0),
    document_id uuid NOT NULL,
    document_updated_at timestamptz NOT NULL,
    review_id uuid,
    approval_id uuid,
    PRIMARY KEY (job_id,ordinal),
    UNIQUE (job_id,document_id),
    FOREIGN KEY (organization_id,job_id) REFERENCES document_export_snapshots(organization_id,job_id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id)
);
CREATE INDEX document_review_queue ON documents (organization_id,status,accepted_at,id)
    WHERE status IN ('Available','Archived');
ALTER TABLE document_review_drafts ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_review_draft_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_review_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_review_returns ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_export_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_export_snapshot_rows ENABLE ROW LEVEL SECURITY;
CREATE POLICY review_drafts_read ON document_review_drafts FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_review_drafts.organization_id
        AND d.id=document_review_drafts.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
          OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY review_drafts_write ON document_review_drafts FOR ALL USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_review_drafts.organization_id
        AND d.id=document_review_drafts.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted) OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))))
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_review_drafts.organization_id
        AND d.id=document_review_drafts.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted) OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY review_history_read ON document_review_draft_history FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND (updated_by=app_user_id() OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_review_draft_history.organization_id
        AND d.id=document_review_draft_history.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted) OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY review_history_insert ON document_review_draft_history FOR INSERT WITH CHECK
    (organization_id=app_organization_id() AND updated_by=app_user_id()
      AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_review_draft_history.organization_id
        AND d.id=document_review_draft_history.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
          OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY review_tasks_read ON document_review_tasks FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_review_tasks.organization_id
        AND d.id=document_review_tasks.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
          OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY review_tasks_write ON document_review_tasks FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY review_returns_manager ON document_review_returns FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY review_returns_assignee_read ON document_review_returns FOR SELECT USING
    (organization_id=app_organization_id() AND EXISTS (SELECT 1 FROM document_review_tasks rt
      JOIN tasks t ON t.organization_id=rt.organization_id AND t.id=rt.task_id
      JOIN documents d ON d.organization_id=rt.organization_id AND d.id=rt.document_id
      WHERE rt.organization_id=document_review_returns.organization_id
        AND rt.document_id=document_review_returns.document_id
        AND t.assignee_user_id=app_user_id() AND t.status IN ('Open','InProgress')
        AND d.status IN ('Available','Archived') AND
          ((d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
            OR d.assignee_user_id=app_user_id()
            OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
              AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY document_export_snapshots_owner ON document_export_snapshots FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY document_export_rows_owner ON document_export_snapshot_rows FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));

-- Match the existing Document deep-link rule for sensitive confirmed and approved rows.
DROP POLICY document_extraction_read ON document_extraction_reviews;
CREATE POLICY document_extraction_read ON document_extraction_reviews FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_extraction_reviews.organization_id
        AND d.id=document_extraction_reviews.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted) OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
DROP POLICY accounting_suggestions_read ON accounting_suggestions;
CREATE POLICY accounting_suggestions_read ON accounting_suggestions FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=accounting_suggestions.organization_id
        AND d.id=accounting_suggestions.document_id AND d.status IN ('Available','Archived')
        AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
          OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted) OR d.assignee_user_id=app_user_id()
          OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
            AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));

CREATE FUNCTION purge_document_review_values() RETURNS trigger LANGUAGE plpgsql
SECURITY DEFINER SET search_path = public, pg_temp AS $$
BEGIN
    IF NEW.status='Purged' AND OLD.status<>'Purged' THEN
        DELETE FROM document_review_returns WHERE organization_id=NEW.organization_id AND document_id=NEW.id;
        DELETE FROM document_review_draft_history WHERE organization_id=NEW.organization_id AND document_id=NEW.id;
        DELETE FROM document_review_drafts WHERE organization_id=NEW.organization_id AND document_id=NEW.id;
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER document_review_values_purge AFTER UPDATE OF status ON documents
    FOR EACH ROW EXECUTE FUNCTION purge_document_review_values();
