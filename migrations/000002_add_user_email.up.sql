BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE public.users ADD COLUMN email varchar(254) NOT NULL DEFAULT '';
COMMIT;
