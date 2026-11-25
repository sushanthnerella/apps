-- Add active custom offerings to discovery without replacing existing services.
WITH missing AS (
    SELECT DISTINCT ON (cs.priest_id, lower(btrim(cs.title)))
           cs.priest_id, btrim(cs.title) AS title
    FROM priest_custom_services cs
    JOIN priests p ON p.id = cs.priest_id
    WHERE cs.is_active = TRUE
      AND btrim(cs.title) <> ''
      AND NOT EXISTS (
          SELECT 1 FROM unnest(COALESCE(p.services_offered, ARRAY[]::text[])) AS offered(name)
          WHERE lower(btrim(offered.name)) = lower(btrim(cs.title))
      )
    ORDER BY cs.priest_id, lower(btrim(cs.title)), btrim(cs.title)
), additions AS (
    SELECT priest_id, array_agg(title ORDER BY lower(title), title) AS titles
    FROM missing
    GROUP BY priest_id
)
UPDATE priests p
SET services_offered = COALESCE(p.services_offered, ARRAY[]::text[]) || additions.titles
FROM additions
WHERE p.id = additions.priest_id;
