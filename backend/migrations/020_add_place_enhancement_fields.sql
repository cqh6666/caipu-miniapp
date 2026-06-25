ALTER TABLE places ADD COLUMN revisit_rating INTEGER NOT NULL DEFAULT 0;
ALTER TABLE places ADD COLUMN recommended_items_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE places ADD COLUMN phone TEXT NOT NULL DEFAULT '';
ALTER TABLE places ADD COLUMN external_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE places ADD COLUMN external_poi_id TEXT NOT NULL DEFAULT '';
ALTER TABLE places ADD COLUMN rating TEXT NOT NULL DEFAULT '';
ALTER TABLE places ADD COLUMN dining_tips TEXT NOT NULL DEFAULT '';
ALTER TABLE places ADD COLUMN scenes_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE places ADD COLUMN best_time TEXT NOT NULL DEFAULT '';
ALTER TABLE places ADD COLUMN duration TEXT NOT NULL DEFAULT '';
ALTER TABLE places ADD COLUMN companion_tags_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE places ADD COLUMN parking_note TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_places_revisit_rating
  ON places(kitchen_id, revisit_rating, updated_at);

CREATE INDEX IF NOT EXISTS idx_places_external_poi
  ON places(external_provider, external_poi_id);
