package depman

import (
	"database/sql"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"time"
)

type Library struct {
	Id      int       `json:"library_id"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
}

func (l Library) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(l)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (l Library) ToString() string {
	//TODO
	return ""
}

func (l *Library) GetVersions() (LibraryVersions, error) {
	versions, err := GetLibraryVersionsByLibraryId(l.Id)
	return versions, err
}

func (l *Library) GetVersion(version string) (*LibraryVersion, error) {
	lv, err := GetLibraryVersionByLibraryIdVersion(l.Id, version)
	return lv, err
}

func (l *Library) GetLatestVersion() (*LibraryVersion, error) {
	lv := &LibraryVersion{}
	return lv, nil
}

func GetLibraryById(id int) (*Library, error) {
	l := &Library{}
	return l, nil
}

func GetLibraryByName(name string) (*Library, error) {
	l := &Library{}

	query := "SELECT library_id, name, created FROM libraries WHERE name = $1"

	err := dbconn.QueryRow(query, name).Scan(
		&l.Id,
		&l.Name,
		&l.Created)

	switch {
	case err == sql.ErrNoRows:
		log.Infof("No library with name: %s", name)
		return l, ErrNotFound
	case err != nil:
		log.Error(err)
		return l, err
	}

	return l, nil
}

type LibraryVersion struct {
	Id        int       `json:"library_version_id"`
	LibraryId int       `json:"library_id"`
	Version   string    `json:"version"`
	Created   time.Time `json:"created"`
}

func (l LibraryVersion) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(l)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (l LibraryVersion) ToString() string {
	//TODO
	return ""
}

func GetLibraryVersionsByLibraryId(library_id int) (LibraryVersions, error) {
	versions := LibraryVersions{}

	query := `SELECT library_version_id, library_id, version, created
		FROM library_versions
		WHERE library_id = $1`

	log.Debugf("Query: %s", query)

	rows, err := dbconn.Query(query, library_id)
	if err != nil {
		return versions, err
	}

	for rows.Next() {
		lv := LibraryVersion{}
		rows.Scan(&lv.Id, &lv.LibraryId, &lv.Version, &lv.Created)

		versions = append(versions, lv)
	}

	return versions, err
}

func GetLibraryVersionByLibraryIdVersion(library_id int, version string) (*LibraryVersion, error) {
	lv := &LibraryVersion{}

	query := `SELECT library_version_id, library_id, version, created
		FROM library_versions
		WHERE library_id = $1 AND version = $2`

	log.Debugf("Query: %s", query)

	err := dbconn.
		QueryRow(query, library_id, version).
		Scan(&lv.Id, &lv.LibraryId, &lv.Version, &lv.Created)

	switch {
	case err == sql.ErrNoRows:
		return lv, ErrNotFound
	case err != nil:
		log.Error(err)
		return lv, err
	}

	return lv, nil

}

func (lv *LibraryVersion) GetFilesByFilter(filter map[string]interface{}) (Files, error) {
	filter["library_version_id"] = lv.Id
	files, err := GetFilesByFilter(filter)

	switch {
	case err != nil:
		return files, err
	case len(files) == 0:
		return files, ErrNotFound
	default:
		return files, err
	}
}

func GetLibraryVersionById(id int) (*LibraryVersion, error) {
	lv := &LibraryVersion{}
	return lv, nil
}

func GetLibraryVersionByIdVersion(id int, version string) (*LibraryVersion, error) {
	lv := &LibraryVersion{}
	return lv, nil
}

func GetLibraryVersionByNameVersion(libname string, version string) (*LibraryVersion, error) {
	lv := &LibraryVersion{}
	return lv, nil
}

type LibraryVersions []LibraryVersion

func (l LibraryVersions) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(l)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (l LibraryVersions) ToString() string {
	//TODO
	return ""
}
