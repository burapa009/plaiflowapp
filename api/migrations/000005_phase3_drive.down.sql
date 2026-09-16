DROP FUNCTION IF EXISTS mark_drive_reconnect_required(uuid,bigint,timestamptz,uuid);
DROP FUNCTION IF EXISTS consume_drive_oauth_attempt(bytea,timestamptz);
DROP TABLE IF EXISTS drive_reconnect_tasks;
DROP TABLE IF EXISTS drive_connections;
DROP TABLE IF EXISTS drive_oauth_attempts;
