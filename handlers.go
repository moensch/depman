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
	"time"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome!")
}

func HandleListLibraries(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Listing libraries")
}

func HandleGetLibrary(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Get Library")

	log.Debugf("Load library: %s", reqVars["libname"])

	lib, err := GetLibraryByName(reqVars["libname"])

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

func HandleGetLibraryVersions(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Listing versions by libname")

	lib, err := GetLibraryByName(reqVars["libname"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	versions, err := lib.GetVersions()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, versions)
}

func getLibVer(libname string, libver string) (*LibraryVersion, error) {
	lv := &LibraryVersion{}

	lib, err := GetLibraryByName(libname)

	if err != nil {
		return lv, err
	}

	lv, err = lib.GetVersion(libver)

	if err != nil {
		return lv, err
	}

	return lv, err
}

func HandleGetLibraryVersion(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Get Library Version")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, lv)
}

func HandleGetLibraryFiles(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "List Files")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatform(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "List Files")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArch(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "List Files")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
		"arch":     reqVars["arch"],
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArchType(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "List Files")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
		"arch":     reqVars["arch"],
		"type":     reqVars["filetype"],
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArchTypeName(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "List Files")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
		"arch":     reqVars["arch"],
		"type":     reqVars["filetype"],
		"name":     reqVars["filename"],
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArchTypeNameLinks(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Get Links")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
		"arch":     reqVars["arch"],
		"type":     reqVars["filetype"],
		"name":     reqVars["filename"],
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	links, err := files[0].GetLinks()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, links)
}

func HandleAddLink(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "Add Link")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
		"arch":     reqVars["arch"],
		"type":     reqVars["filetype"],
		"name":     reqVars["filename"],
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	fl := &FileLink{}
	fl.FileId = files[0].Id
	fl.Name = reqVars["linkname"]
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

	var lib *Library
	var lv *LibraryVersion
	lib, err := GetLibraryByName(reqVars["libname"])

	switch {
	case err != nil && err == ErrNotFound:
		// Create Library
		lib.Name = reqVars["libname"]
		err = lib.Store()
		if err != nil {
			SendErrorResponse(w, r, err)
		}
		log.Debugf("Created new library with ID: %d", lib.Id)
	case err != nil:
		SendErrorResponse(w, r, err)
	}

	lv, err = lib.GetVersion(reqVars["libver"])
	switch {
	case err != nil && err == ErrNotFound:
		// Create LibraryVersion
		lv.LibraryId = lib.Id
		lv.Version = reqVars["libver"]
		err = lv.Store()
		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
		log.Debugf("Created new libver with ID: %d", lv.Id)
	case err != nil:
		SendErrorResponse(w, r, err)
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
		"arch":     reqVars["arch"],
		"type":     reqVars["filetype"],
		"name":     reqVars["filename"],
	})

	var file File

	switch {
	case err == ErrNotFound:
		// Create the file in the database
		log.Debug("File not found, storing")
		file = File{0, lv.Id, reqVars["filename"], reqVars["filetype"], reqVars["platform"], reqVars["arch"], time.Now(), FileLinks{}, "", ""}
		err = file.Store()
		if err != nil {
			SendErrorResponse(w, r, err)
			return
		}
	case err != nil:
		SendErrorResponse(w, r, err)
		return
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

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": reqVars["platform"],
		"arch":     reqVars["arch"],
		"type":     reqVars["filetype"],
		"name":     reqVars["filename"],
	})

	if err != nil {
		SendErrorResponse(w, r, err)
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

func HandlePutLibraryFilesPlatformArchTypeName(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	logRequest(reqVars, "File Store")

	lv, err := getLibVer(reqVars["libname"], reqVars["libver"])

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	file := &File{0, lv.Id, reqVars["filename"], reqVars["filetype"], reqVars["platform"], reqVars["arch"], time.Now(), FileLinks{}, "", ""}

	err = file.Store()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, file)
}
