package depman

import (
	"encoding/json"
	"strings"
)

type JsonAble interface {
	ToJsonString() (string, error)
	ToString() string
}

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
	return l.Name
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
	entries := make([]string, len(l))
	for idx, e := range l {
		entries[idx] = e.ToString()
	}

	return strings.Join(entries, "\n")
}
