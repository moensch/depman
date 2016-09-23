package depman

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome!")
}

func HandleListLibraries(w http.ResponseWriter, r *http.Request) {
	libraries, err := ListLibraries()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, libraries)
}

func HandleGetLibrary(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Get Library")

	lib, err := GetLibrary(reqVars["library"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, lib)
}

func logRequest(muxvars map[string]string, msg string) {
	f := log.Fields{}
	for k, v := range muxvars {
		f[k] = v
	}
	log.WithFields(f).Info(msg)
}

func HandleListLibraryVersions(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Listing versions by library")

	versions, err := ListVersions(reqVars["library"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, versions)
}

func HandleGetLibraryVersion(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Get Library Version")

	lv, err := GetVersion(reqVars["library"], reqVars["version"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, lv)
}

func reqToFilter(req map[string]string) map[string]interface{} {
	filter := make(map[string]interface{})
	for k, v := range req {
		filter[k] = v
	}
	return filter
}

func HandleListFiles(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "List Files")

	files, err := GetFilesByFilter(reqToFilter(reqVars), true)

	switch {
	case err != nil && err == ErrNotFound:
		fallthrough
	case err == nil && len(files) == 0:
		log.Debugf("No files found - try default namespace %s", DefaultNS)
		reqVars["ns"] = DefaultNS
		files, err = GetFilesByFilter(reqToFilter(reqVars), true)

		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
	case err != nil:
		SendErrorResponse(w, r, err)
		return
	}

	if len(files) == 0 {
		SendErrorResponse(w, r, ErrNotFound)
	} else {
		SendResponse(w, r, files)
	}
}

func HandleGetFileLinks(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Get Links")

	files, err := GetFilesByFilter(reqToFilter(reqVars), true)
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	if len(files) == 0 {
		SendErrorResponse(w, r, ErrNotFound)
		return
	}

	links, err := files[0].GetLinks()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, links)
}

func HandlePutLink(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Add Link")

	linkname := reqVars["linkname"]
	delete(reqVars, "linkname")
	files, err := GetFilesByFilter(reqToFilter(reqVars), false)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	if len(files) == 0 {
		SendErrorResponse(w, r, ErrNotFound)
		return
	}

	links, err := files[0].GetLinks()

	for _, link := range links {
		if link.Name == linkname {
			log.Debugf("File link already exists - not creating")

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Stored")
			return
		}
	}

	fl := &FileLink{}
	fl.FileId = files[0].Id
	fl.Name = linkname
	err = fl.Store()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Stored")
}

func HandleFileUpload(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "File Upload")

	files, err := GetFilesByFilter(reqToFilter(reqVars), false)

	var file File

	switch {
	case err != nil:
		SendErrorResponse(w, r, err)
		return
	case len(files) == 0:
		// Create the file in the database
		log.Debug("File not found, storing")
		file = NewFileFromVars(reqVars)
		err = file.Store()
		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
	default:
		log.Debugf("Found file: %d", files[0].Id)
		file = files[0]
	}

	log.Infof("Storing file at %s", file.FilePath())

	_, err = os.Stat(file.FilePath())
	if err == nil {
		log.Debug("File exists - removing")
		os.Remove(file.FilePath())
	}

	err = os.MkdirAll(filepath.Dir(file.FilePath()), 0700)
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	localfile, err := os.OpenFile(file.FilePath(), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	for {
		buffer := make([]byte, 4096)
		len, err := r.Body.Read(buffer)
		if err != nil && err != io.EOF {
			SendErrorResponse(w, r, err)
			return
		}
		if len == 0 {
			// Nothing more to read
			log.Debug("Finished reading")
			break
		}
		log.Debugf("Read %d bytes from request body", len)
		_ = bytes.Trim(buffer, "\x00")
		len_w, err := localfile.Write(buffer[:len])
		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
		log.Debugf("Wrote %d bytes to file", len_w)
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Stored")
}

func HandleFileDownload(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "File Download")

	files, err := GetFilesByFilter(reqToFilter(reqVars), true)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}
	if len(files) == 0 {
		SendErrorResponse(w, r, ErrNotFound)
		return
	}

	file := files[0]
	fh, err := os.Open(file.FilePath())
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	buffer := make([]byte, 4096)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	for {
		len, err := fh.Read(buffer)
		if err != nil && err != io.EOF {
			SendErrorResponse(w, r, err)
			return
		}
		if len == 0 {
			// Nothing more to read
			log.Debug("Finished reading")
			break
		}
		log.Debugf("Read %d bytes from disk", len)
		len_w, err := w.Write(buffer[:len])
		if len_w != len {
			SendErrorResponse(w, r, errors.New(fmt.Sprintf("Only wrote %d bytes, but should have written %d", len_w, len)))
			return
		}
		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
		log.Debugf("Wrote %d bytes to http response", len_w)
	}
}

func HandlePutFile(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "File Store")

	file := NewFileFromVars(reqVars)

	err := file.Store()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, file)
}

func HandleGetExtraFile(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Get Extrafile")

	file, err := GetExtraFileByFilter(reqToFilter(reqVars))

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, file)
}

func HandleDownloadExtraFile(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Download Extrafile")

	file, err := GetExtraFileByFilter(reqToFilter(reqVars))
	switch {
	case err != nil && err == ErrNotFound:
		log.Debugf("No files found - try default namespace %s", DefaultNS)
		reqVars["ns"] = DefaultNS
		file, err = GetExtraFileByFilter(reqToFilter(reqVars))

		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
	case err != nil:
		SendErrorResponse(w, r, err)
		return
	}

	fh, err := os.Open(file.FilePath())
	defer fh.Close()
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	written, err := io.Copy(w, fh)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}
	log.Debugf("Wrote %d bytes", written)
}

func HandleUploadExtraFile(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Upload Extrafile")

	file, err := GetExtraFileByFilter(reqToFilter(reqVars))

	switch {
	case err != nil && err == ErrNotFound:
		// Create the file in the database
		log.Debug("File not found, storing")
		file = NewExtraFileFromVars(reqVars)
		err = file.Store()
		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
	case err != nil:
		SendErrorResponse(w, r, err)
		return
	default:
		log.Debugf("Found file: %d", file.Id)
	}

	log.Infof("Storing file at %s", file.FilePath())

	_, err = os.Stat(file.FilePath())
	if err == nil {
		log.Debug("File exists - removing")
		os.Remove(file.FilePath())
	}

	err = os.MkdirAll(filepath.Dir(file.FilePath()), 0700)
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	localfile, err := os.OpenFile(file.FilePath(), os.O_WRONLY|os.O_CREATE, 0600)
	defer localfile.Close()
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	written, err := io.Copy(localfile, r.Body)
	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}
	log.Debugf("Wrote %d bytes", written)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Stored")
}
