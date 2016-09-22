package depman

import (
	"database/sql"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strings"
	"time"
)

type File struct {
	Id               int       `json:"file_id"`
	LibraryVersionId int       `json:"library_version_id"`
	Library          string    `json:"library"`
	Version          string    `json:"version"`
	NameSpace        string    `json:"ns"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Platform         string    `json:"platform"`
	Arch             string    `json:"arch"`
	Info             string    `json:"info"`
	Created          time.Time `json:"created"`
	Links            FileLinks `json:"file_links"`
}

func (f File) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(f)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (f File) ToString() string {
	return fmt.Sprintf("%s/%s/%s/%s", f.Platform, f.Arch, f.Type, f.Name)
}

func NewFileFromVars(vars map[string]string) File {
	f := File{}

	for k, v := range vars {
		switch k {
		case "library":
			f.Library = v
		case "version":
			f.Version = v
		case "ns":
			f.NameSpace = v
		case "name":
			f.Name = v
		case "type":
			f.Type = v
		case "platform":
			f.Platform = v
		case "arch":
			f.Arch = v
		}
	}

	return f
}

type Files []File

func (f Files) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(f)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (f Files) ToString() string {
	entries := make([]string, len(f))
	for idx, e := range f {
		entries[idx] = e.ToString()
	}

	return strings.Join(entries, "\n")
}

func GetLatestVersion(filter map[string]interface{}, table string) (string, error) {
	// Don't filter by version anymore
	ver_search := filter["version"]
	delete(filter, "version")
	query := fmt.Sprintf("SELECT version FROM %s WHERE ", table)

	// TODO: Code duplication with the below
	var i int = 0
	var values = make([]interface{}, len(filter))
	var where_clauses = make([]string, len(filter))
	for col, val := range filter {
		where_clauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
		values[i] = val
		i++
	}

	if ver_search != "latest" {
		// Search by prefix - for gigglz
		where_clauses = append(where_clauses, fmt.Sprintf("version LIKE $%d || '%%'", len(values)+1))
		values = append(values, ver_search)
	}

	query += strings.Join(where_clauses, " AND ")
	query += " ORDER BY string_to_array(version, '.')::int[] DESC LIMIT 1"
	log.Debugf("GetLatestVersion query: %s", query)

	version := ""

	err := dbconn.QueryRow(query, values...).Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		return version, ErrNotFound
	case err != nil:
		log.Error(err)
		return version, err
	}

	log.Debugf("Latest version: %s", version)

	return version, err
}

func GetFilesByFilter(filter map[string]interface{}) (Files, error) {
	files := Files{}

	if _, ok := filter["version"]; ok {
		// Got version
		log.Debug("Have to find latest version")
		ver, err := GetLatestVersion(filter, "files")
		if err == nil {
			filter["version"] = ver
		}
	}
	query := `SELECT file_id, library, version, ns, name, type, platform, arch, info, created
		FROM files
		WHERE `

	// TODO: Code duplication with the above
	var i int = 0
	var values = make([]interface{}, len(filter))
	var where_clauses = make([]string, len(filter))
	for col, val := range filter {
		where_clauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
		values[i] = val
		i++
	}

	query = query + strings.Join(where_clauses, " AND ")
	log.Debugf("Filterquery: %s", query)

	rows, err := dbconn.Query(query, values...)
	if err != nil {
		return files, err
	}

	for rows.Next() {
		file := File{}
		rows.Scan(&file.Id, &file.Library, &file.Version, &file.NameSpace, &file.Name, &file.Type, &file.Platform, &file.Arch, &file.Info, &file.Created)

		files = append(files, file)
	}

	for idx, _ := range files {
		files[idx].Links, _ = files[idx].GetLinks()
	}

	return files, nil
}

func (f *File) GetLinks() (FileLinks, error) {
	links, err := GetFileLinksByFileId(f.Id)
	return links, err
}

func (f *File) Store() error {
	var query string
	if f.Id == 0 {
		//insert
		query = `INSERT INTO files (library, version, ns, name, type, platform, arch, info)
			VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING file_id
			`
	} else {
		//update
		query = `UPDATE files SET library=$1, version=$2, ns=$3, name=$4, type=$5, platform=$6, arch=$7 WHERE file_id = $8`
	}

	var lastInsertId int
	err := dbconn.QueryRow(query, f.Library, f.Version, f.NameSpace, f.Name, f.Type, f.Platform, f.Arch, f.Info).Scan(&lastInsertId)
	if err != nil {
		return err
	}
	log.Debugf("Stored as: %d", lastInsertId)
	f.Id = lastInsertId

	return nil
}

func (f *File) FilePath() string {
	return fmt.Sprintf(StoreDir+"/%s/%s/%s/%s/%s/%s/%s", f.NameSpace, f.Library, f.Version, f.Platform, f.Arch, f.Type, f.Name)
}

func (fl *FileLink) Store() error {
	var query string
	if fl.Id == 0 {
		//insert
		query = `INSERT INTO filelinks (file_id, name)
			VALUES
			($1, $2)
			RETURNING file_link_id
			`
	} else {
		//update
		query = `UPDATE filelinks SET file_id=$1, name=$2 WHERE file_link_id = $3`
	}

	var lastInsertId int
	err := dbconn.QueryRow(query, fl.FileId, fl.Name).Scan(&lastInsertId)
	if err != nil {
		return err
	}
	log.Debugf("Stored as: %d", lastInsertId)
	fl.Id = lastInsertId

	return nil
}
