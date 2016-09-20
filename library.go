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
	Ns      string    `json:"ns"`
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

func (l *Library) Store() error {
	var query string
	if l.Id == 0 {
		//insert
		query = `INSERT INTO libraries (name, ns)
			VALUES
			($1, $2)
			RETURNING library_id
			`
	} else {
		//update
		query = `UPDATE libraries SET name=$1, ns=$2 WHERE library_id = $3`
	}

	var lastInsertId int
	err := dbconn.QueryRow(query, l.Name, l.Ns).Scan(&lastInsertId)
	if err != nil {
		return err
	}
	log.Debugf("Stored as: %d", lastInsertId)
	l.Id = lastInsertId

	return nil
}

func (l *Library) GetVersions() (LibraryVersions, error) {
	versions, err := GetLibraryVersionsByLibraryId(l.Id)
	return versions, err
}

func (l *Library) GetVersion(version string) (*LibraryVersion, error) {
	if version == "latest" {
		return l.GetLatestVersion()
	} else {
		lv, err := GetLibraryVersionByLibraryIdVersion(l.Id, version)
		return lv, err
	}
}

func (l *Library) GetLatestVersion() (*LibraryVersion, error) {
	lv, err := GetLatestLibraryVersion(l.Id)
	return lv, err
}

func GetLibraryByName(name string) (*Library, error) {
	l := &Library{}

	query := "SELECT library_id, name, ns, created FROM libraries WHERE name = $1"

	err := dbconn.QueryRow(query, name).Scan(
		&l.Id,
		&l.Name,
		&l.Ns,
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

func GetLatestLibraryVersion(library_id int) (*LibraryVersion, error) {
	lv := &LibraryVersion{}

	query := `SELECT library_version_id, library_id, version, created
		FROM library_versions
		WHERE library_id = $1
		ORDER BY string_to_array(version, '.')::int[] DESC
		LIMIT 1`

	log.Debugf("Query: %s", query)

	err := dbconn.
		QueryRow(query, library_id).
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

func GetLibraryVersionByLibraryIdVersion(library_id int, version string) (*LibraryVersion, error) {
	lv := &LibraryVersion{}

	query := `SELECT library_version_id, library_id, version, created
		FROM library_versions
		WHERE library_id = $1 AND version LIKE $2 || '%'
		ORDER BY string_to_array(version, '.')::int[] DESC
		LIMIT 1`

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

func (lv *LibraryVersion) Store() error {
	var query string
	if lv.Id == 0 {
		//insert
		query = `INSERT INTO library_versions (library_id, version)
			VALUES
			($1, $2)
			RETURNING library_version_id
			`
	} else {
		//update
		query = `UPDATE library_versions SET library_id=$1, version=$2 WHERE library_version_id = $3`
	}

	var lastInsertId int
	err := dbconn.QueryRow(query, lv.LibraryId, lv.Version).Scan(&lastInsertId)
	if err != nil {
		return err
	}
	log.Debugf("Stored as: %d", lastInsertId)
	lv.Id = lastInsertId

	return nil
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
