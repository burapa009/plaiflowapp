package postgres

import (
	"context"

	"plaiflow/api/internal/matching"
	"plaiflow/api/internal/tenant"
)

func (s *Store) FindMatchCandidates(ctx context.Context, user, org string, source matching.Facts) ([]matching.Facts, error) {
	if source.DocumentID == "" || source.TotalAmount == "" || source.IssueDate == "" {
		return []matching.Facts{}, nil
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, user, org); err != nil {
		return nil, tenant.ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT i.document_id,i.document_type,to_char(i.issue_date,'YYYY-MM-DD'),
		i.total_amount::text,i.document_number,i.seller_tax_id,i.seller_name,
		CASE WHEN $5::text<>'' AND i.seller_name<>'' THEN similarity(lower(i.seller_name),lower($5::text)) ELSE 0 END
		FROM document_match_index i
		JOIN documents d ON d.organization_id=i.organization_id AND d.id=i.document_id
		JOIN document_extraction_reviews r ON r.organization_id=i.organization_id AND r.id=i.review_id
		JOIN document_ocr_runs o ON o.organization_id=i.organization_id AND o.job_id=r.ocr_job_id
		WHERE i.organization_id=$1 AND i.document_id<>$2 AND d.status IN ('Available','Archived')
		AND (organization_role($6::uuid,$1::uuid) IN ('Owner','Admin')
		  OR (d.submitted_by_user_id=$6::uuid AND NOT d.group_restricted) OR d.assignee_user_id=$6::uuid
		  OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
		    AND ds.document_id=d.id AND ds.submitted_by_user_id=$6::uuid AND NOT ds.group_source))
		AND r.superseded_at IS NULL AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
		AND i.total_amount BETWEEN $3::numeric-0.01 AND $3::numeric+0.01
		AND i.issue_date BETWEEN $4::date-3 AND $4::date+3
		ORDER BY abs(i.total_amount-$3::numeric),abs(i.issue_date-$4::date),i.document_id LIMIT 50`,
		org, source.DocumentID, source.TotalAmount, source.IssueDate, source.SellerName, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]matching.Facts, 0, 10)
	for rows.Next() {
		var item matching.Facts
		if err := rows.Scan(&item.DocumentID, &item.DocumentType, &item.IssueDate, &item.TotalAmount,
			&item.DocumentNumber, &item.SellerTaxID, &item.SellerName, &item.VendorSimilarity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}
