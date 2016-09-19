package depman

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"time"
)

type FileLink struct {
	Id      int       `json:"file_link_id"`
	FileId  int       `json:"file_id"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
}

type FileLinks []FileLink

func GetFileLinksByFileId(file_id int) (FileLinks, error) {
	links := FileLinks{}

	query := `SELECT file_link_id, file_id, name, created
		FROM filelinks
		WHERE file_id = $1`

	log.Debugf("Query: %s", query)

	rows, err := dbconn.Query(query, file_id)
	if err != nil {
		return links, err
	}

	for rows.Next() {
		fl := FileLink{}
		rows.Scan(&fl.Id, &fl.FileId, &fl.Name, &fl.Created)

		links = append(links, fl)
	}

	return links, err
}

func (f FileLinks) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(f)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (f FileLinks) ToString() string {
	//TODO
	return ""
}
