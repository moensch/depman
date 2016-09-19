package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/moensch/depman"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	depmanUrl      string
	depmanPlatform string
	depmanArch     string
	logLevel       string
	includeDir     string
	libDir         string
	depFile        string
	httpClient     *http.Client
)

func init() {
	flag.StringVar(&depmanArch, "a", "", "Architecture (e.g. 'x86_64')")
	flag.StringVar(&depmanPlatform, "p", "", "Platform (e.g. 'el6')")
	flag.StringVar(&depmanUrl, "s", os.Getenv("DEPMAN_URL"), "Server URL")
	flag.StringVar(&includeDir, "i", "include/", "Include dir")
	flag.StringVar(&libDir, "l", "lib/", "Lib dir")
	flag.StringVar(&logLevel, "d", "info", "Log level (debug|info|warn|error")
	flag.StringVar(&depFile, "f", "depman_deps.txt", "Dependency file")
}

func main() {
	flag.Parse()

	lvl, _ := log.ParseLevel(logLevel)
	log.SetLevel(lvl)

	includeDir = strings.TrimSuffix(includeDir, "/")
	libDir = strings.TrimSuffix(libDir, "/")
	depmanUrl = strings.TrimSuffix(depmanUrl, "/")
	log.Infof("Architecture: %s", depmanArch)
	log.Infof("Platform    : %s", depmanPlatform)
	log.Infof("Include Dir : %s", includeDir)
	log.Infof("Library Dir : %s", libDir)
	log.Infof("Dependencies: %s", depFile)

	for _, dir := range []string{includeDir, libDir} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				log.Fatalf("ERROR: %s")
			}
		}
	}

	httpClient = &http.Client{}

	fh, err := os.Open(depFile)
	if err != nil {
		log.Fatalf("Cannot open dependency file %s: %s", depFile, err)
	}

	defer fh.Close()

	r := bufio.NewReader(fh)

	re := regexp.MustCompile("(\\S+):(\\S+)")
	line, _, err := r.ReadLine()
	for err == nil {
		s := string(line)
		log.Debugf("Read line: '%s'", s)
		line, _, err = r.ReadLine()

		matches := re.FindAllStringSubmatch(s, -1)
		var libname string
		var libver string

		switch {
		case len(matches) == 0:
			log.Fatalf("Line in dep file cannot be parsed: %s", line)
		default:
			libname = matches[0][1]
			libver = matches[0][2]
		}

		log.Infof("Want lib: %s / %s", libname, libver)

		downloadLib(libname, libver)
		/*
			if err != nil {
				log.Fatalf("ERROR: %s", err)
			}
		*/
	}
}

func downloadLib(libname string, libver string) error {
	body, err := GETRequestJSON(fmt.Sprintf("/lib/%s/versions/%s/files/%s/%s", libname, libver, depmanPlatform, depmanArch))

	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	files := depman.Files{}

	if err = json.Unmarshal(body, &files); err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	// headers: 0644
	// SO: 0755
	// Archives: 0644
	for _, file := range files {
		log.Infof("Downloading: %s %s", file.Type, file.Name)

		var mode os.FileMode
		var dir string
		switch file.Type {
		case "header":
			dir = includeDir
			mode = 0644
		case "archive":
			dir = libDir
			mode = 0644
		case "shared":
			dir = libDir
			mode = 0755
		}

		err = downloadFile(libname, libver, file, dir, mode)
		if err != nil {
			// TODO: Make fatal once done testing
			log.Warnf("ERROR: %s", err)
			continue
		}

		localfile := dir + "/" + file.Name
		for _, link := range file.Links {
			symlink := dir + "/" + link.Name
			log.Infof("  Symlink: %s => %s", symlink, localfile)

			err = os.Symlink(file.Name, symlink)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downloadFile(libname string, libver string, f depman.File, dir string, mode os.FileMode) error {
	tmpfile := dir + "/" + "." + f.Name + ".dwn"
	localfile := dir + "/" + f.Name

	log.Infof("Downloading %s/%s/%s to %s", libname, libver, f.Name, localfile)

	for _, fname := range []string{tmpfile, localfile} {
		_, err := os.Stat(fname)
		if err == nil {
			log.Debugf("File %s exists - removing", fname)
			os.Remove(fname)
		}
	}

	path := fmt.Sprintf("/lib/%s/versions/%s/files/%s/%s/%s/%s/download",
		libname, libver, depmanPlatform, depmanArch, f.Type, f.Name)

	log.Debugf("Download URL: %s", path)

	req_url := strings.Join([]string{depmanUrl, path}, "")

	req, err := http.NewRequest("GET", req_url, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return err
	}

	fh, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE, mode)
	if err != nil {
		return err
	}
	for {
		buffer := make([]byte, 4096)
		len, err := resp.Body.Read(buffer)

		if err != nil && err != io.EOF {
			fh.Close()
			os.Remove(tmpfile)
			return err
		}
		if len == 0 {
			// Nothing more to read
			log.Debug("Finished reading")
			break
		}

		log.Debugf("Read %d bytes", len)

		len_w, err := fh.Write(buffer[:len])
		if err != nil {
			fh.Close()
			os.Remove(tmpfile)
			return err
		}
		log.Debugf("Wrote %d bytes", len_w)
	}

	fh.Close()

	err = os.Rename(tmpfile, localfile)

	return err
}

func GETRequestJSON(path string) ([]byte, error) {
	return GETRequest(path, "application/json")
}

func GETRequest(path string, accept string) ([]byte, error) {
	req_url := strings.Join([]string{depmanUrl, path}, "")

	req, err := http.NewRequest("GET", req_url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	l := log.WithFields(log.Fields{"url": req_url, "httpcode": resp.StatusCode, "method": "GET"})
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		l.Warn("HTTP error")
		return nil, errors.New(fmt.Sprintf("HTTP Error %d", resp.StatusCode))
	}
	l.Debug("HTTP log")

	return ioutil.ReadAll(resp.Body)

}
