-- Emergency application rollback should disable Phase 7 routes and retain data.
-- Run this destructive schema rollback only after explicitly preserving review artifacts.
DROP TABLE IF EXISTS document_extraction_reviews;
