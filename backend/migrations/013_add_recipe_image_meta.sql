ALTER TABLE recipes ADD COLUMN image_meta_json TEXT NOT NULL DEFAULT '[]';

UPDATE recipes
SET image_meta_json = COALESCE((
  SELECT json_group_array(
    json_object(
      'url', trimmed_url,
      'sourceType', 'legacy',
      'originUrl', trimmed_url
    )
  )
  FROM (
    SELECT TRIM(COALESCE(json_each.value, '')) AS trimmed_url
    FROM json_each(
      CASE
        WHEN COALESCE(TRIM(image_urls_json), '') <> '' AND TRIM(image_urls_json) <> '[]' THEN image_urls_json
        WHEN COALESCE(TRIM(image_url), '') <> '' THEN json_array(image_url)
        ELSE '[]'
      END
    )
    WHERE TRIM(COALESCE(json_each.value, '')) <> ''
  )
), '[]')
WHERE COALESCE(TRIM(image_meta_json), '') = '' OR TRIM(image_meta_json) = '[]';
