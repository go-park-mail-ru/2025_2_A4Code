-- +migrate Down
DROP TRIGGER IF EXISTS appeal_update_trigger ON appeal;
DROP TABLE IF EXISTS appeal CASCADE;

DROP TYPE IF EXISTS appeal_status CASCADE;