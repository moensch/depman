package depman

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"strconv"
	"strings"
	"time"
)

type File struct {
	Id               int       `json:"file_id"`
	LibraryVersionId int       `json:"library_version_id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Platform         string    `json:"platform"`
	Arch             string    `json:"arch"`
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
	//TODO
	return ""
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
	//TODO
	return ""
}

func GetFilesByFilter(filter map[string]interface{}) (Files, error) {
	files := Files{}

	query := `SELECT file_id, library_version_id, name, type, platform, arch, created
		FROM files
		WHERE `

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
		rows.Scan(&file.Id, &file.LibraryVersionId, &file.Name, &file.Type, &file.Platform, &file.Arch, &file.Created)

		files = append(files, file)
	}

	for idx, _ := range files {
		files[idx].Links, _ = files[idx].GetLinks()
	}

	return files, err
}

func (f *File) GetLinks() (FileLinks, error) {
	links, err := GetFileLinksByFileId(f.Id)
	return links, err
}

func (f *File) Store() error {
	var query string
	if f.Id == 0 {
		//insert
		query = `INSERT INTO files (library_version_id, name, type, platform, arch)
			VALUES
			($1, $2, $3, $4, $5)
			RETURNING file_id
			`
	} else {
		//update
		query = `UPDATE files SET library_version_id=$1, name=$2, type=$3, platform=$4, arch=$5 WHERE file_id = $6`
	}

	var lastInsertId int
	err := dbconn.QueryRow(query, f.LibraryVersionId, f.Name, f.Type, f.Platform, f.Arch).Scan(&lastInsertId)
	if err != nil {
		return err
	}
	log.Debugf("Stored as: %d", lastInsertId)
	f.Id = lastInsertId

	return nil
}

func (f *File) Stream(out io.Writer) error {
	return nil
}

func (f *File) FilePath() string {
	return "/tmp/depman_files" + "/" + strconv.Itoa(f.Id)
}
