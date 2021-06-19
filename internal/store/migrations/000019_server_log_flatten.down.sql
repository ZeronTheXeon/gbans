begin;

ALTER TABLE IF EXISTS server_log DROP COLUMN IF EXISTS weapon;
ALTER TABLE IF EXISTS server_log DROP COLUMN IF EXISTS damage;
ALTER TABLE IF EXISTS server_log DROP COLUMN IF EXISTS attacker_position;
ALTER TABLE IF EXISTS server_log DROP COLUMN IF EXISTS victim_position;
ALTER TABLE IF EXISTS server_log DROP COLUMN IF EXISTS assister_position;
ALTER TABLE IF EXISTS server_log DROP COLUMN IF EXISTS item;

commit;
