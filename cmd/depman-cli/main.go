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
	"os/exec"
	"path/filepath"
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
	flag.StringVar(&depmanArch, "a", "", "Architecture (e.g. 'x86_64'. Default: uname -m)")
	flag.StringVar(&depmanPlatform, "p", "", "Platform (e.g. 'el6'. Default: Read from rpm --eval %dist)")
	flag.StringVar(&depmanUrl, "s", os.Getenv("DEPMAN_URL"), "Server URL")
	flag.StringVar(&includeDir, "i", "depman-include/", "Include dir")
	flag.StringVar(&libDir, "l", "depman-lib/", "Lib dir")
	flag.StringVar(&logLevel, "d", "info", "Log level (debug|info|warn|error|fatal)")
	flag.StringVar(&depFile, "f", "depman_deps.txt", "Dependency file")
}

func getArch() string {
	for _, uname := range []string{"/usr/bin/uname", "/bin/uname"} {
		_, err := os.Stat(uname)
		if err != nil {
			continue
		}
		out, err := exec.Command(uname, "-m").Output()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		re := regexp.MustCompile("(x86_64|i386)")

		matches := re.FindAllStringSubmatch(string(out), -1)
		if len(matches) == 0 {
			return ""
		}

		return matches[0][1]
	}

	return ""
}

func getPlatform() string {
	for _, rpm := range []string{"/usr/bin/rpm", "/bin/rpm"} {
		out, err := exec.Command(rpm, "--eval", "%dist").Output()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		re := regexp.MustCompile("(el5|el6|el7)")

		matches := re.FindAllStringSubmatch(string(out), -1)
		if len(matches) == 0 {
			return ""
		}

		return matches[0][1]
	}

	return ""
}

func main() {
	flag.Parse()

	lvl, _ := log.ParseLevel(logLevel)
	log.SetLevel(lvl)

	if depmanArch == "" {
		depmanArch = getArch()
	}
	if depmanPlatform == "" {
		depmanPlatform = getPlatform()
	}

	if depmanArch == "" {
		log.Fatal("Cannot determine architecture - use -a to provide it")
	}

	if depmanPlatform == "" {
		log.Fatal("Cannot determine platform - use -p to provide it")
	}

	includeDir = strings.TrimSuffix(includeDir, "/")
	libDir = strings.TrimSuffix(libDir, "/")
	depmanUrl = strings.TrimSuffix(depmanUrl, "/")

	if flag.NArg() < 1 {
		log.Warnf("Not enough parameters")
		flag.Usage()
		os.Exit(1)
	}

	operation := flag.Arg(0)
	log.Infof("Architecture: %s", depmanArch)
	log.Infof("Platform    : %s", depmanPlatform)
	log.Infof("Include Dir : %s", includeDir)
	log.Infof("Library Dir : %s", libDir)
	log.Infof("Dependencies: %s", depFile)

	if operation == "get" {
		for _, dir := range []string{includeDir, libDir} {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err = os.MkdirAll(dir, 0755)
				if err != nil {
					log.Fatalf("ERROR: %s")
				}
			}
		}
	}

	httpClient = &http.Client{}

	switch operation {
	case "get":
		libnames, err := downloadFilesFromDepfile(depFile)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		makeFlags(libnames)
	case "upload":
		log.Infof("Uploading...")
		if flag.NArg() < 4 {
			log.Warnf("Not enough parameters")
			flag.Usage()
			os.Exit(1)
		}
		libname := flag.Arg(1)
		libver := flag.Arg(2)

		files := flag.Args()[3:]
		log.Infof("Library: %s", libname)
		log.Infof("Version: %s", libver)

		err := uploadFiles(libname, libver, files)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}
	}
}

func getFileType(path string) (string, error) {
	log.Debugf("Determining file type for %s (ext: '%s')", path, filepath.Ext(path))
	switch {
	case filepath.Ext(path) == ".h":
		fallthrough
	case filepath.Ext(path) == ".hpp":
		return "header", nil
	case filepath.Ext(path) == ".a":
		return "archive", nil
	case strings.Contains(filepath.Base(path), ".so"):
		return "shared", nil
	default:
		return "", errors.New(fmt.Sprintf("Cannot determine file type for %s using extension %s", path, filepath.Ext(path)))
	}
}

func uploadFiles(libname string, libver string, files []string) error {
	url_tpl := fmt.Sprintf("/lib/%s/versions/%s/files/%s/%s/%%s/%%s/%%s", libname, libver, depmanPlatform, depmanArch)
	log.Debugf("URL: %s", url_tpl)

	//links := make([]string, 0)

	links := make(map[string][]string)

	uploaded_files := make([][]string, 0)
	for _, f := range files {
		filename := filepath.Base(f)
		log.Infof("Considering file: %s", f)
		info, err := os.Lstat(f)
		if err != nil {
			return err
		}

		switch {
		case info.Mode().IsRegular():
			log.Infof("  File %s is a regular file", filename)
			filetype, err := getFileType(f)
			if err != nil {
				log.Warn(err)
				continue
			}
			err = uploadFile(f, fmt.Sprintf(url_tpl, filetype, filename, "upload"))
			if err != nil {
				return err
			}

			log.Infof("  Successfully uploaded %s (type: %s)", filename, filetype)
			uploaded_files = append(uploaded_files, []string{filename, filetype})
		case info.Mode()&os.ModeSymlink == os.ModeSymlink:
			//target, _ := os.Readlink(f)
			target, _ := filepath.EvalSymlinks(f)
			targetfile := filepath.Base(target)
			log.Infof("  File %s is a symlink to %s", filename, targetfile)

			if _, ok := links[targetfile]; ok {
				// append
				links[targetfile] = append(links[targetfile], filename)
			} else {
				links[targetfile] = make([]string, 0)
				links[targetfile] = append(links[targetfile], filename)
			}
		}
	}

	for _, f := range uploaded_files {
		if _, ok := links[f[0]]; ok {
			log.Infof("Processing symlinks for: %s", f[0])
			for _, linkname := range links[f[0]] {
				log.Infof("  Linkname: %s", linkname)
				path := fmt.Sprintf(url_tpl+"/%s", f[1], f[0], "links", linkname)
				log.Debugf("  Linkpath: %s", path)

				req, err := http.NewRequest("PUT", strings.Join([]string{depmanUrl, path}, ""), nil)
				if err != nil {
					return err
				}
				resp, err := httpClient.Do(req)
				defer resp.Body.Close()
				if err != nil {
					return err
				}
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					return errors.New(fmt.Sprintf("Http error: %d", resp.StatusCode))
				}
			}
		}
	}
	return nil
}

func uploadFile(localfile string, path string) error {
	req_url := strings.Join([]string{depmanUrl, path}, "")

	stat, err := os.Stat(localfile)
	if err != nil {
		return err
	}
	log.Debugf("Uploading %s to %s", localfile, req_url)
	log.Debugf("Uploading %d bytes", stat.Size())

	fh, err := os.Open(localfile)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", req_url, fh)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New(fmt.Sprintf("Http error: %d", resp.StatusCode))
	}

	return nil
}

func downloadFilesFromDepfile(depfile string) ([]string, error) {
	libnames := make([]string, 0)
	fh, err := os.Open(depfile)
	if err != nil {
		return libnames, errors.New(fmt.Sprintf("Cannot open dependency file %s: %s", depfile, err))
	}

	defer fh.Close()

	r := bufio.NewReader(fh)

	re := regexp.MustCompile("(\\S+):(\\S+)")

	for err == nil {
		line, _, err := r.ReadLine()
		if err != nil && err != io.EOF {
			return libnames, err
		} else if err == io.EOF {
			break
		}
		s := string(line)
		log.Debugf("Read line: '%s'", s)

		matches := re.FindAllStringSubmatch(s, -1)
		var libname string
		var libver string

		switch {
		case len(matches) == 0:
			return libnames, errors.New(fmt.Sprintf("Line in dep file cannot be parsed: %s", line))
		default:
			libname = matches[0][1]
			libver = matches[0][2]
		}

		log.Infof("Want lib: %s / %s", libname, libver)

		err = downloadLib(libname, libver)
		if err != nil {
			return libnames, err
		}

		libnames = append(libnames, strings.TrimPrefix(libname, "lib"))
	}

	return libnames, nil
}

func makeFlags(libnames []string) {
	cflags := fmt.Sprintf("-I%s", includeDir)
	ldflags := fmt.Sprintf("-L%s ", libDir)

	ldflags = ldflags + "-l" + strings.Join(libnames, " -l")

	fmt.Printf("CCFLAGS = %s\n", cflags)
	fmt.Printf("CCLDFLAGS = %s\n", ldflags)
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

			os.Remove(symlink)
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

	log.Infof("Download URL: %s", path)

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
		return errors.New(fmt.Sprintf("Http error: %d", resp.StatusCode))
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
