ALTER TABLE kitchens
  ADD COLUMN is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0, 1));

WITH ranked_kitchens AS (
  SELECT id,
         ROW_NUMBER() OVER (
           PARTITION BY owner_user_id
           ORDER BY CASE WHEN COALESCE(name_source, 'custom') = 'auto' THEN 0 ELSE 1 END,
                    id ASC
         ) AS owner_rank
    FROM kitchens
)
UPDATE kitchens
   SET is_default = 1
 WHERE id IN (
   SELECT id
     FROM ranked_kitchens
    WHERE owner_rank = 1
 );

CREATE UNIQUE INDEX idx_kitchens_one_default_per_owner
  ON kitchens(owner_user_id)
  WHERE is_default = 1;

ALTER TABLE recipes
  ADD COLUMN version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1);

ALTER TABLE places
  ADD COLUMN version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1);
