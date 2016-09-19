package depman

import (
	"database/sql"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"net/http"
)

var (
	dbconn   *sql.DB
	StoreDir string
)

var ErrNotFound = errors.New("Entry not found")

type DepMan struct {
	Router *mux.Router
}

func NewServer() (*DepMan, error) {
	depman := &DepMan{}
	var err error
	// TODO: Pull from config or some such
	dbconn, err = sql.Open("postgres", "user=depman password=depman dbname=depman host=127.0.0.1 port=5432 sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	depman.Router = NewRouter()
	log.Info("initialized")
	return depman, nil
}

func (c *DepMan) Run(listenAddr string) {
	log.Infof("Listening on: %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, c.Router))
}

func prepareFilter(pairs ...string) (map[string]string, error) {
	m, err := mapFromPairsToString(pairs...)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// checkPairs returns the count of strings passed in, and an error if
// the count is not an even number.
func checkPairs(pairs ...string) (int, error) {
	length := len(pairs)
	if length%2 != 0 {
		return length, fmt.Errorf(
			"mux: number of parameters must be multiple of 2, got %v", pairs)
	}
	return length, nil
}

// mapFromPairsToString converts variadic string parameters to a
// string to string map.
func mapFromPairsToString(pairs ...string) (map[string]string, error) {
	length, err := checkPairs(pairs...)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, length/2)
	for i := 0; i < length; i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m, nil
}
