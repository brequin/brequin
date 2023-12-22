-- AND-DEPS ARE MODELED AS SEPARATE EDGES,
-- OR-DEPS ARE MODELED AS ONE EDGE TO A SWITCH NODE
-- THAT HAS SEPARATE EDGES TO VALUE (COURSE) NODES

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
  'value', 'switch'
);

CREATE TABLE nodes (
  id text PRIMARY KEY,
  type node_type NOT NULL
);

CREATE TABLE courses (
  node_id text UNIQUE NOT NULL REFERENCES nodes(id),
  subject_area_code text REFERENCES subject_areas(code),
  course_number text,
  course_description text,
  PRIMARY KEY (subject_area_code, course_number)
);

CREATE TYPE grade AS ENUM (
  'A+', 'A', 'A-', 'B+', 'B', 'B-', 'C+', 'C', 'C-', 'D+', 'D', 'D-', 'F'
);

CREATE TABLE relations (
  source_id text NOT NULL REFERENCES nodes(id),
  target_id text NOT NULL REFERENCES nodes(id),
  enforced boolean,
  prereq boolean,
  coreq boolean,
  minimum_grade grade,
  PRIMARY KEY (source_id, target_id)
);
