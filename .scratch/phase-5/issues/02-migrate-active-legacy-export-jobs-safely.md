# 02: Migrate active legacy export jobs safely

**What to build:** Existing active background exports enter the Durable Job model without duplicate execution or lost history, while the previous export history remains readable and rollback remains possible throughout the migration window.

**Blocked by:** 01: Worker claims one Durable Job exactly once.

**Status:** ready-for-agent

- [ ] The expand migration adds the new job model without dropping or changing the meaning of existing export history.
- [ ] Existing Queued and Running background exports migrate with stable identifiers, Organization, requester, filters, format, row count and safe timestamps.
- [ ] A legacy Running row without a valid new lease becomes reclaimable instead of being trusted as actively owned.
- [ ] Migrated exports cannot be executed by both legacy and Durable Job workers.
- [ ] Existing completed/failed export history remains readable through a compatibility read path.
- [ ] New producers switch to Durable Jobs only after the new schema and Internal Job API are available.
- [ ] The down/rollback path stops new Durable Job creation and preserves readable export history and artifact references.
- [ ] No destructive removal of the legacy table or compatibility path occurs in this ticket.
- [ ] Migration tests cover empty, queued, running and terminal legacy datasets and prove that rollback does not strand accepted work.
