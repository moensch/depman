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
	depmanNs       string
	depmanPlatform string
	depmanArch     string
	logLevel       string
	includeDir     string
	libDir         string
	depFile        string
	httpClient     *http.Client
)

func init() {
	flag.StringVar(&depmanNs, "n", "", "Name space")
	flag.StringVar(&depmanArch, "a", "", "Architecture (e.g. 'x86_64'. Default: uname -m)")
	flag.StringVar(&depmanPlatform, "p", "", "Platform (e.g. 'el6'. Default: Read from rpm --eval %dist)")
	flag.StringVar(&depmanUrl, "s", os.Getenv("DEPMAN_URL"), "Server URL")
	flag.StringVar(&includeDir, "i", "depman-include/", "Include dir")
	flag.StringVar(&libDir, "l", "depman-lib/", "Lib dir")
	flag.StringVar(&logLevel, "d", "warn", "Log level (debug|info|warn|error|fatal)")
	flag.StringVar(&depFile, "f", "depman_deps.txt", "Dependency file")
}

func getArch() (string, error) {
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
			return "", fmt.Errorf("Cannot find arch in string '%s' returned by uname -m", out)
		}

		return matches[0][1], nil
	}

	return "", errors.New("Cannot determine architecture - use -a flag")
}

func getPlatform() (string, error) {
	for _, rpm := range []string{"/usr/bin/rpm", "/bin/rpm"} {
		out, err := exec.Command(rpm, "--eval", "%dist").Output()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		re := regexp.MustCompile("(el5|el6|el7)")

		matches := re.FindAllStringSubmatch(string(out), -1)
		if len(matches) == 0 {
			return "", fmt.Errorf("Cannot find platform in string '%s' returned by rpm --eval %%dist", out)
		}

		return matches[0][1], nil
	}

	return "", errors.New("Cannot determine platform - use -a flag")
}

func main() {
	flag.Parse()

	lvl, _ := log.ParseLevel(logLevel)
	log.SetLevel(lvl)
	var err error
	if depmanArch == "" {
		if depmanArch, err = getArch(); err != nil {
			log.Fatal(err)
		}
	}
	if depmanPlatform == "" {
		if depmanPlatform, err = getPlatform(); err != nil {
			log.Fatal(err)
		}
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
	url_tpl := fmt.Sprintf("/v1/lib/%s/%s/versions/%s/files/%s/%s/%%s/%%s/%%s", "ns", libname, libver, depmanPlatform, depmanArch)
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

func parseDepLine(line string) (map[string]string, error) {
	// split by pipe, surrounded by optional white space
	white := regexp.MustCompile("\\s*\\|\\s*")
	fields := white.Split(line, -1)

	ret := map[string]string{
		"library": "",
		"version": "",
		"incdir":  "",
		"wanted":  "header,archive,shared",
	}

	for idx, field := range fields {
		switch {
		case idx == 0:
			parts := strings.Split(field, ":")
			ret["library"] = parts[0]
			if len(parts) > 1 {
				ret["version"] = parts[1]
			} else {
				ret["version"] = "latest"
			}
		case idx == 1:
			ret["incdir"] = strings.TrimSuffix(field, "/")
			if !strings.HasPrefix(ret["incdir"], "/") {
				ret["incdir"] = "/" + ret["incdir"]
			}
		case idx == 2:
			ret["wanted"] = field
		}
	}

	for k, v := range ret {
		log.Debugf("Dependency %s => %s", k, v)
	}
	return ret, nil
}

func downloadFilesFromDepfile(depfile string) ([]string, error) {
	libnames := make([]string, 0)
	fh, err := os.Open(depfile)
	if err != nil {
		return libnames, errors.New(fmt.Sprintf("Cannot open dependency file %s: %s", depfile, err))
	}

	defer fh.Close()

	r := bufio.NewReader(fh)

	for err == nil {
		line, _, err := r.ReadLine()
		if err != nil && err != io.EOF {
			return libnames, err
		} else if err == io.EOF {
			break
		}
		s_line := string(line)
		log.Debugf("Read line: '%s'", s_line)
		if s_line == "" {
			// empty line
			continue
		}
		dep, err := parseDepLine(s_line)

		log.Infof("Want lib: %s / %s in subdir %s", dep["library"], dep["version"], dep["incdir"])

		err = downloadLib(dep["library"], dep["version"], dep["wanted"], dep["incdir"])
		if err != nil {
			return libnames, err
		}

		libnames = append(libnames, strings.TrimPrefix(dep["library"], "lib"))
	}

	return libnames, nil
}

func makeFlags(libnames []string) {
	includeDirAbs, err := filepath.Abs(includeDir)
	if err != nil {
		log.Fatal(err)
	}

	libDirAbs, err := filepath.Abs(libDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("DEPMAN_LIB_DIR = %s\n", libDirAbs)
	fmt.Printf("DEPMAN_INC_DIR = %s\n", includeDirAbs)
	fmt.Printf("DEPMAN_CFLAGS = -I$(DEPMAN_INC_DIR)\n")
	fmt.Printf("DEPMAN_CCLDFLAGS = -L$(DEPMAN_LIB_DIR)")
	for _, lib := range libnames {
		fmt.Printf(" \\\n\t-l%s", lib) // line continuation on previous line plus tab
	}
	fmt.Printf("\n")
}

func downloadLib(libname string, libver string, wanted string, include_subdir string) error {
	body, err := GETRequestJSON(fmt.Sprintf("/v1/lib/%s/%s/versions/%s/files/%s/%s", "ns", libname, libver, depmanPlatform, depmanArch))

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
		switch {
		case file.Type == "header" && strings.Contains(wanted, "header"):
			dir = includeDir
			if include_subdir != "" {
				dir = dir + include_subdir
			}
			mode = 0644
		case file.Type == "archive" && strings.Contains(wanted, "archive"):
			dir = libDir
			mode = 0644
		case file.Type == "shared" && strings.Contains(wanted, "shared"):
			dir = libDir
			mode = 0755
		default:
			log.Warnf("Ignoring file %s of type %s as it is not wanted", file.Name, file.Type)
			continue
		}

		err = downloadFile(libname, libver, file, dir, mode)
		if err != nil {
			return err
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
	tmpfile := fmt.Sprintf("%s/.%s.dwn", dir, f.Name)
	localfile := fmt.Sprintf("%s/%s", dir, f.Name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	log.Infof("Downloading %s/%s/%s to %s", libname, libver, f.Name, localfile)

	for _, fname := range []string{tmpfile, localfile} {
		_, err := os.Stat(fname)
		if err == nil {
			log.Debugf("File %s exists - removing", fname)
			os.Remove(fname)
		}
	}

	path := fmt.Sprintf("/v1/lib/%s/%s/versions/%s/files/%s/%s/%s/%s/download",
		"ns", libname, libver, depmanPlatform, depmanArch, f.Type, f.Name)

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
