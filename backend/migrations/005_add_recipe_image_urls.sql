ALTER TABLE recipes ADD COLUMN image_urls_json TEXT NOT NULL DEFAULT '[]';

UPDATE recipes
SET image_urls_json = '[]'
WHERE image_urls_json IS NULL OR TRIM(image_urls_json) = '';

UPDATE recipes
SET image_urls_json = json_array(image_url)
WHERE COALESCE(TRIM(image_url), '') <> ''
  AND COALESCE(TRIM(image_urls_json), '') = '[]';
