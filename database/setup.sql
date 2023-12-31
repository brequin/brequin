-- AND-DEPS ARE MODELED BY VALUE (COURSE)/AND NODE,
-- OR-DEPS ARE MODELED BY OR NODE

CREATE TABLE quarters (
  code text PRIMARY KEY,
  name text UNIQUE NOT NULL
);

CREATE TABLE subject_areas (
  code text PRIMARY KEY,
  name text UNIQUE NOT NULL
);

CREATE TABLE quarter_subject_areas (
  quarter_code text REFERENCES quarters(code),
  subject_area_code text REFERENCES subject_areas(code),
  PRIMARY KEY (quarter_code, subject_area_code)
);

CREATE TYPE node_type AS ENUM (
  'value', 'and', 'or'
);

CREATE TABLE nodes (
  id text PRIMARY KEY,
  type node_type NOT NULL
);

CREATE TABLE courses (
  subject_area_code text REFERENCES subject_areas(code),
  catalog_number text,
  node_id text UNIQUE NOT NULL REFERENCES nodes(id),
  PRIMARY KEY (subject_area_code, catalog_number)
);

CREATE TABLE courses_details (
  subject_area_code text,
  catalog_number text,
  name text NOT NULL,
  units text NOT NULL,
  level text NOT NULL,
  description text NOT NULL,
  PRIMARY KEY (subject_area_code, catalog_number)
);

CREATE TABLE relations (
  source_id text REFERENCES nodes(id),
  target_id text REFERENCES nodes(id),
  enforced text,
  prereq text,
  coreq text,
  minimum_grade text,
  PRIMARY KEY (source_id, target_id, enforced, prereq, coreq, minimum_grade)
);

CREATE FUNCTION quarter_rank(code text) RETURNS text AS $$
  DECLARE
    year text := LEFT(code, 2);
  BEGIN
    CASE RIGHT(code, 1)
      WHEN 'W' THEN RETURN year || '0';
      WHEN 'S' THEN RETURN year || '1';
      WHEN '1' THEN RETURN year || '2';
      WHEN '2' THEN RETURN year || '3';
      WHEN 'F' THEN RETURN year || '4';
    END CASE;
  END;
$$ LANGUAGE plpgsql;
