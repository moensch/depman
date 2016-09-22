package depman

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"os"
)

func SendResponse(w http.ResponseWriter, r *http.Request, resp JsonAble) {
	switch r.Header.Get("Accept") {
	case "text/plain":
		stringresp := resp.ToString()
		SendTEXTResponse(w, http.StatusOK, stringresp)
	default:
		stringresp, _ := resp.ToJsonString()
		SendJSONResponse(w, http.StatusOK, stringresp)
	}
}

func SendErrorResponse(w http.ResponseWriter, r *http.Request, err_resp error) {

	var code int

	switch {
	case err_resp == ErrNotFound:
		fallthrough
	case os.IsNotExist(err_resp):
		code = http.StatusNotFound
	default:
		code = http.StatusInternalServerError
	}

	log.WithFields(log.Fields{
		"error": err_resp,
		"code":  code,
	}).Error("HTTP ERROR")

	switch r.Header.Get("Accept") {
	case "text/plain":
		SendTEXTResponse(w, code, err_resp.Error())
	default:

		resp := struct {
			Error string `json:"error"`
		}{
			err_resp.Error(),
		}
		jsonblob, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s\n", err)
			return
		}
		SendJSONResponse(w, code, string(jsonblob))
	}
}

func SendTEXTResponse(w http.ResponseWriter, code int, resp string) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "%s\n", resp)
}

func SendJSONResponse(w http.ResponseWriter, code int, resp string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	fmt.Fprint(w, resp)
}
