package depman

import (
	"database/sql"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strings"
	"time"
)

type ExtraFile struct {
	Id        int       `json:"extrafile_id"`
	Version   string    `json:"version"`
	NameSpace string    `json:"ns"`
	Name      string    `json:"name"`
	Info      string    `json:"info"`
	Created   time.Time `json:"created"`
}

func (f ExtraFile) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(f)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (f ExtraFile) ToString() string {
	return fmt.Sprintf("%s/%s/%s", f.NameSpace, f.Version, f.Name)
}

func GetExtraFileByFilter(filter map[string]interface{}) (ExtraFile, error) {
	ef := ExtraFile{}

	if _, ok := filter["version"]; ok {
		// Got version
		log.Debug("Have to find latest version")
		ver, err := GetLatestVersion(filter, "extrafiles")
		if err == nil {
			filter["version"] = ver
		}
	}

	query := `SELECT extrafile_id, version, ns, name, info, created
		FROM extrafiles
		WHERE version=$1 AND ns=$2 AND name=$3`

	log.Debugf("Query: %s", query)
	err := dbconn.QueryRow(query,
		filter["version"],
		filter["ns"],
		filter["name"]).
		Scan(&ef.Id, &ef.Version, &ef.NameSpace, &ef.Name, &ef.Info, &ef.Created)

	switch {
	case err == sql.ErrNoRows:
		return ef, ErrNotFound
	case err != nil:
		log.Error(err)
		return ef, err
	}

	return ef, err
}

func NewExtraFileFromVars(vars map[string]string) ExtraFile {
	f := ExtraFile{}

	for k, v := range vars {
		switch k {
		case "version":
			f.Version = v
		case "ns":
			f.NameSpace = v
		case "name":
			f.Name = v
		}
	}

	return f
}

func (f *ExtraFile) FilePath() string {
	return fmt.Sprintf(StoreDir+"/_extras_/%s/%s/%s", f.NameSpace, f.Name, f.Version)
}

func (f *ExtraFile) Store() error {
	var query string
	if f.Id == 0 {
		//insert
		query = `INSERT INTO extrafiles (version, ns, name, info)
			VALUES
			($1, $2, $3, $4)
			RETURNING extrafile_id
			`
	} else {
		//update
		query = `UPDATE files SET library=$1, version=$2, ns=$3, name=$4, type=$5, platform=$6, arch=$7 WHERE file_id = $8`
	}

	var lastInsertId int
	err := dbconn.QueryRow(query, f.Version, f.NameSpace, f.Name, f.Info).Scan(&lastInsertId)
	if err != nil {
		return err
	}
	log.Debugf("Stored as: %d", lastInsertId)
	f.Id = lastInsertId

	return nil
}

type ExtraFiles []ExtraFile

func (f ExtraFiles) ToJsonString() (string, error) {
	var retval string
	jsonblob, err := json.Marshal(f)
	if err != nil {
		return retval, err
	}
	return string(jsonblob), err
}

func (f ExtraFiles) ToString() string {
	entries := make([]string, len(f))
	for idx, e := range f {
		entries[idx] = e.ToString()
	}

	return strings.Join(entries, "\n")
}
