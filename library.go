package depman

import (
	"database/sql"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
)

type SimpleEntry struct {
	Name string `json:"name"`
}

func (l SimpleEntry) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(l)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (l SimpleEntry) ToString() string {
	//TODO
	return ""
}

type SimpleEntries []SimpleEntry

func (l SimpleEntries) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(l)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (l SimpleEntries) ToString() string {
	//TODO
	return ""
}

func ListLibraries() (SimpleEntries, error) {
	entries := SimpleEntries{}

	query := "SELECT DISTINCT(library) FROM files"
	log.Debugf("Query: %s", query)

	rows, err := dbconn.Query(query)
	if err != nil {
		return entries, err
	}

	for rows.Next() {
		entry := SimpleEntry{}
		rows.Scan(&entry.Name)

		entries = append(entries, entry)
	}

	return entries, err
}

func GetLibrary(library string) (SimpleEntry, error) {
	entry := SimpleEntry{}

	query := "SELECT library FROM files WHERE library = $1"
	log.Debugf("Query: %s", query)

	err := dbconn.
		QueryRow(query, library).
		Scan(&entry.Name)

	switch {
	case err == sql.ErrNoRows:
		return entry, ErrNotFound
	case err != nil:
		log.Error(err)
		return entry, err
	}

	return entry, nil
}

func ListVersions(library string) (SimpleEntries, error) {
	entries := SimpleEntries{}

	query := "SELECT DISTINCT(version) FROM files WHERE library = $1"
	log.Debugf("Query: %s", query)

	rows, err := dbconn.Query(query, library)
	if err != nil {
		return entries, err
	}

	for rows.Next() {
		entry := SimpleEntry{}
		rows.Scan(&entry.Name)

		entries = append(entries, entry)
	}

	return entries, err
}

func GetVersion(library string, version string) (SimpleEntry, error) {
	entry := SimpleEntry{}

	query := "SELECT version FROM files WHERE library = $1 and version LIKE $2 || '%' ORDER BY string_to_array(version, '.')::int[] DESC LIMIT 1"
	log.Debugf("Query: %s", query)

	err := dbconn.
		QueryRow(query, library, version).
		Scan(&entry.Name)

	switch {
	case err == sql.ErrNoRows:
		return entry, ErrNotFound
	case err != nil:
		log.Error(err)
		return entry, err
	}

	return entry, nil
}
