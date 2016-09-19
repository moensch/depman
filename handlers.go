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
	libname := reqVars["libname"]

	log.Debugf("Load library: %s", libname)

	lib, err := GetLibraryByName(libname)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, lib)
}

func HandleGetLibraryVersions(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	libname := reqVars["libname"]

	log.WithFields(log.Fields{
		"library": libname,
	}).Info("Listing versions by name")

	lib, err := GetLibraryByName(libname)

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
	libname := reqVars["libname"]
	libver := reqVars["libver"]

	log.WithFields(log.Fields{
		"library": libname,
		"version": libver,
	}).Info("Getting library version")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, lv)
}

func HandleGetLibraryFiles(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	libname := reqVars["libname"]
	libver := reqVars["libver"]

	log.WithFields(log.Fields{
		"library": libname,
		"version": libver,
	}).Info("Listing files by name")

	lv, err := getLibVer(libname, libver)

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
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
	}).Info("Listing files by platform")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": platform,
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArch(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
	}).Info("Listing files by platform, arch")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": platform,
		"arch":     arch,
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArchType(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]
	filetype := reqVars["filetype"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
		"filetype": filetype,
	}).Info("Listing files by platform, arch, filetype")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": platform,
		"arch":     arch,
		"type":     filetype,
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArchTypeName(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]
	filetype := reqVars["filetype"]
	filename := reqVars["filename"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
		"filetype": filetype,
		"filename": filename,
	}).Info("Listing files by platform, arch, filetype, filename")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": platform,
		"arch":     arch,
		"type":     filetype,
		"name":     filename,
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, files)
}

func HandleGetLibraryFilesPlatformArchTypeNameLinks(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]
	filetype := reqVars["filetype"]
	filename := reqVars["filename"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
		"filetype": filetype,
		"filename": filename,
	}).Info("Listing links for file by platform, arch, filetype, filename")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": platform,
		"arch":     arch,
		"type":     filetype,
		"name":     filename,
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
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]
	filetype := reqVars["filetype"]
	filename := reqVars["filename"]
	linkname := reqVars["linkname"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
		"filetype": filetype,
		"filename": filename,
		"linkname": linkname,
	}).Info("Add Link")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": platform,
		"arch":     arch,
		"type":     filetype,
		"name":     filename,
	})

	if err != nil {
		SendErrorResponse(w, r, err)
		return
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
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]
	filetype := reqVars["filetype"]
	filename := reqVars["filename"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
		"filetype": filetype,
		"filename": filename,
	}).Info("File Upload")

	var lib *Library
	var lv *LibraryVersion
	lib, err := GetLibraryByName(libname)

	switch {
	case err != nil && err == ErrNotFound:
		// Create Library
		lib.Name = libname
		err = lib.Store()
		if err != nil {
			SendErrorResponse(w, r, err)
		}
		log.Debugf("Created new library with ID: %d", lib.Id)
	case err != nil:
		SendErrorResponse(w, r, err)
	}

	lv, err = lib.GetVersion(libver)
	switch {
	case err != nil && err == ErrNotFound:
		// Create LibraryVersion
		lv.LibraryId = lib.Id
		lv.Version = libver
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
		"platform": platform,
		"arch":     arch,
		"type":     filetype,
		"name":     filename,
	})

	var file File

	switch {
	case err == ErrNotFound:
		// Create the file in the database
		log.Debug("File not found, storing")
		file = File{0, lv.Id, filename, filetype, platform, arch, time.Now(), FileLinks{}, "", ""}
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
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]
	filetype := reqVars["filetype"]
	filename := reqVars["filename"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
		"filetype": filetype,
		"filename": filename,
	}).Info("File Download")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	files, err := lv.GetFilesByFilter(map[string]interface{}{
		"platform": platform,
		"arch":     arch,
		"type":     filetype,
		"name":     filename,
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
	libname := reqVars["libname"]
	libver := reqVars["libver"]
	platform := reqVars["platform"]
	arch := reqVars["arch"]
	filetype := reqVars["filetype"]
	filename := reqVars["filename"]

	log.WithFields(log.Fields{
		"library":  libname,
		"version":  libver,
		"platform": platform,
		"arch":     arch,
		"filetype": filetype,
		"filename": filename,
	}).Info("File Store")

	lv, err := getLibVer(libname, libver)

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	file := &File{0, lv.Id, filename, filetype, platform, arch, time.Now(), FileLinks{}, "", ""}

	err = file.Store()

	if err != nil {
		SendErrorResponse(w, r, err)
		return
	}

	SendResponse(w, r, file)
}
