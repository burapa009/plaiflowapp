-- Application rollback leaves the stricter policy in place. Restoring the
-- previous FOR ALL policy would reopen direct Member writes. The up is idempotent.
SELECT 1;
