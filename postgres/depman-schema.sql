CREATE TABLE libraries (
  "library_id" SERIAL PRIMARY KEY,
  "name" character varying(255) NOT NULL,
  "ns" character varying(255) NOT NULL,
  "created" timestamp with time zone DEFAULT ('now'::text)::timestamp(6) with time zone
);


CREATE UNIQUE INDEX libraries_name_idx ON libraries(name);


CREATE TABLE library_versions (
  "library_version_id" SERIAL PRIMARY KEY,
  "library_id" integer NOT NULL REFERENCES libraries(library_id) ON DELETE CASCADE,
  "version" character varying(255) NOT NULL,
  "created" timestamp with time zone DEFAULT ('now'::text)::timestamp(6) with time zone
);

CREATE INDEX library_versions_library_id_idx ON library_versions(library_id);

CREATE TYPE filetype AS ENUM ('header', 'archive', 'shared');

CREATE TABLE files (
  "file_id" SERIAL PRIMARY KEY,
  "library_version_id" integer NOT NULL REFERENCES libraries(library_id) ON DELETE CASCADE,
  "name" character varying(255) NOT NULL,
  "type" filetype,
  "platform" character varying(10),
  "arch" character varying(10),
  "created" timestamp with time zone DEFAULT ('now'::text)::timestamp(6) with time zone
);

CREATE INDEX files_library_version_id_idx ON files(library_version_id);
CREATE UNIQUE INDEX files_type_platform_arch_name_idx ON files(library_version_id, type, platform, arch, name);

CREATE TABLE filelinks (
  "file_link_id" SERIAL PRIMARY KEY,
  "file_id" integer NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
  "name" character varying(255) NOT NULL,
  "created" timestamp with time zone DEFAULT ('now'::text)::timestamp(6) with time zone
);

CREATE INDEX filelinks_file_id_idx ON filelinks(file_id);
CREATE UNIQUE INDEX filelinks_file_name_idx ON filelinks(file_id, name);

