
CREATE TYPE filetype AS ENUM ('header', 'archive', 'shared');

CREATE TABLE files (
  "file_id" SERIAL PRIMARY KEY,
  "library" character varying(255) NOT NULL,
  "version" character varying(255) NOT NULL,
  "ns" character varying(255) NOT NULL,
  "name" character varying(255) NOT NULL,
  "type" filetype,
  "platform" character varying(10),
  "arch" character varying(10),
  "info" text NOT NULL DEFAULT '',
  "created" timestamp with time zone DEFAULT ('now'::text)::timestamp(6) with time zone
);

CREATE UNIQUE INDEX files_unique_idx ON files(library, version, ns, name, type, platform, arch);

CREATE TABLE filelinks (
  "file_link_id" SERIAL PRIMARY KEY,
  "file_id" integer NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
  "name" character varying(255) NOT NULL,
  "created" timestamp with time zone DEFAULT ('now'::text)::timestamp(6) with time zone
);

CREATE INDEX filelinks_file_id_idx ON filelinks(file_id);
CREATE UNIQUE INDEX filelinks_file_name_idx ON filelinks(file_id, name);


CREATE TABLE extrafiles (
  "extrafile_id" SERIAL PRIMARY KEY,
  "version" character varying(255) NOT NULL,
  "ns" character varying(255) NOT NULL,
  "name" character varying(255) NOT NULL,
  "info" text NOT NULL DEFAULT '',
  "created" timestamp with time zone DEFAULT ('now'::text)::timestamp(6) with time zone
);

CREATE UNIQUE INDEX extrafiles_unique_idx ON extrafiles(name, ns, version);
