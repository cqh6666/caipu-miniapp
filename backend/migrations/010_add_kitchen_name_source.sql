ALTER TABLE kitchens ADD COLUMN name_source TEXT NOT NULL DEFAULT 'custom';

UPDATE kitchens
SET name_source = 'auto'
WHERE COALESCE(TRIM(name), '') = '我们的厨房';
