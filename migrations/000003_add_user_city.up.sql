BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE public.users ADD COLUMN city varchar(120) NOT NULL DEFAULT '';
COMMIT;
