-- Write your migrate up statements here
ALTER TABLE users DROP COLUMN IF EXISTS email;

---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
