-- Add snapshot_ref column to store the object store key for execution snapshots.
ALTER TABLE workflows ADD COLUMN snapshot_ref VARCHAR(512);
