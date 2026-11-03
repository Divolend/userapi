BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE public.users DROP COLUMN city;
COMMIT;
