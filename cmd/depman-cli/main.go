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
	depmanUrl           string
	depmanNs            string
	depmanPlatform      string
	depmanArch          string
	logLevel            string
	includeDir          string
	libDir              string
	depFile             string
	httpClient          *http.Client
	found_hash_includes map[string]string
	new_header_files    map[string]int
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

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage for %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n\nOperation:\n")
		fmt.Fprintf(os.Stderr, "  get:\n")
		fmt.Fprintf(os.Stderr, "    Pull dependencies from depfile (-f)\n")
		fmt.Fprintf(os.Stderr, "  scan:\n")
		fmt.Fprintf(os.Stderr, "    Recursively scan directory for #includes and pull them\n")
		fmt.Fprintf(os.Stderr, "  search <headerfile>:\n")
		fmt.Fprintf(os.Stderr, "    Find out which library provides a given header file\n")
		fmt.Fprintf(os.Stderr, "  upload <libname> <libver> [list of files...]:\n")
		fmt.Fprintf(os.Stderr, "    Store new binaries and headers (guesses file types from extensions)\n")
		fmt.Fprintf(os.Stderr, "  uploadextra <name> <version> <filepath>:\n")
		fmt.Fprintf(os.Stderr, "    Store an arbitrary file\n")
		fmt.Fprintf(os.Stderr, "  getextra <name> [<version>]:\n")
		fmt.Fprintf(os.Stderr, "    Download extra (non-lib-related) file\n")
		fmt.Fprintf(os.Stderr, "\n\nConfig:\n")
		flag.PrintDefaults()
	}
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
	if depmanNs == "" {
		if depmanNs, err = getNamespace(); err != nil {
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
	log.Infof("Namespace   : %s", depmanNs)
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

	new_header_files = make(map[string]int)
	switch operation {
	case "search":
		if flag.NArg() < 1 {
			log.Warnf("Not enough parameters")
			flag.Usage()
			os.Exit(1)
		}
		searchfile := flag.Arg(1)

		log.Infof("Trying to find which library provides file: %s", searchfile)
		library, version, err := findLibFromFile(searchfile)
		if err != nil {
			log.Fatalf("Not found: %s", err)
		}
		fmt.Printf("%s:%s\n", library, version)
	case "scan":
		ScanWantedTypes := "header,archive,shared"
		if flag.NArg() > 1 {
			ScanWantedTypes = flag.Arg(1)
		}
		log.Warnf("Want: %s", ScanWantedTypes)
		deps := RequiredLibs{}

		// Enter scan loop at current directory
		new_header_files["."] = 1

		var libnames []string

		// Looping to scan newly downloaded files
		for {
			if len(new_header_files) == 0 {
				log.Infof("Nothing more to fetch")
				break
			}

			// Re-assign locally and clear because ParseSourceFiles
			//  appends to new_header_files global again
			files_to_scan := new_header_files
			new_header_files = make(map[string]int)
			for scanfile, _ := range files_to_scan {
				log.Infof("Scanning recursively: %s", scanfile)
				// populates new_header_files and adds to deps
				err := ParseSourceFiles(scanfile, ScanWantedTypes, &deps)
				if err != nil {
					log.Fatalf("ERROR: %s", err)
				}
				// Will skip stuff already downloaded, so it's safe to
				//   call this over and over again
				newlibs, err := deps.Download()
				if err != nil {
					log.Fatalf("ERROR: %s", err)
				}
				for _, l := range newlibs {
					libnames = append(libnames, l)
				}
			}
		}

		makeFlags(libnames)
	case "getextra":
		if flag.NArg() < 2 {
			log.Warnf("Not enough parameters")
			flag.Usage()
			os.Exit(1)
		}

		extrafiles := flag.Args()[1:]
		for _, extrafile := range extrafiles {
			split := strings.Split(extrafile, ":")
			localfile := split[0]
			version := "latest"
			if len(split) == 2 {
				version = split[1]
			}

			log.Infof("Downloading extrafile: %s (%s)", localfile, version)

			uri_path := fmt.Sprintf("/v1/%s/extra/%s/%s/download", depmanNs, localfile, version)

			err = doDownload(uri_path, localfile, 0664)
			if err != nil {
				log.Fatalf("Cannot download extra file: %s", err)
			}
			fmt.Fprintf(os.Stdout, "Downloaded: %s (%s)\n", localfile, version)
		}
	case "get":
		deps, err := ParseDepfile(depFile)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}
		libnames, err := deps.Download()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		makeFlags(libnames)
	case "uploadextra":
		if flag.NArg() < 4 {
			log.Warnf("Not enough parameters")
			flag.Usage()
			os.Exit(1)
		}
		extraname := flag.Arg(1)
		extraver := flag.Arg(2)
		localfile := flag.Arg(3)
		uri_upload := fmt.Sprintf("/v1/%s/extra/%s/%s/upload", depmanNs, extraname, extraver)
		err := uploadFile(localfile, uri_upload)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}
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
	default:
		log.Warnf("Unknown operation: %s", operation)
		flag.Usage()
		os.Exit(1)
	}
}

func getArch() (string, error) {
	for _, uname := range []string{"/usr/bin/uname", "/bin/uname"} {
		_, err := os.Stat(uname)
		if err != nil {
			continue
		}
		out, err := exec.Command(uname, "-m").Output()
		if err != nil {
			log.Warnf("ERROR: %s", err)
			continue
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
			log.Warnf("ERROR: %s", err)
			continue
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

func getNamespace() (string, error) {
	if _, err := os.Stat("/depman_ns.sh"); err == nil {
		log.Debugf("Getting namespace from /depman_ns.sh script")

		out, err := exec.Command("/depman_ns.sh").Output()
		if err != nil {
			return "", fmt.Errorf("Cannot run `/depman_ns.sh': %s", err)
		}

		return strings.TrimSpace(string(out)), nil
	}

	out, err := exec.Command("/usr/bin/hg", "branch").Output()
	if err != nil {
		return "", fmt.Errorf("Cannot run `hg branch': %s", err)
	}

	hg_branch := strings.TrimSpace(string(out))

	out, err = exec.Command("/usr/bin/whoami").Output()
	if err != nil {
		return "", fmt.Errorf("Cannot run `whoami': %s", err)
	}
	username := strings.TrimSpace(string(out))

	return fmt.Sprintf("%s-%s", username, hg_branch), nil
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
	case filepath.Ext(path) == ".o":
		return "object", nil
	case strings.Contains(filepath.Base(path), ".so"):
		return "shared", nil
	default:
		return "", errors.New(fmt.Sprintf("Cannot determine file type for %s using extension %s", path, filepath.Ext(path)))
	}
}

func uploadFiles(libname string, libver string, files []string) error {
	url_tpl := fmt.Sprintf("/v1/%s/lib/%s/versions/%s/files/%s/%s/%%s/%%s/%%s", depmanNs, libname, libver, depmanPlatform, depmanArch)
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

type RequiredLib struct {
	Name       string
	Version    string
	IncDir     string
	Wanted     string
	Downloaded bool
	HasLib     bool
}

func (r *RequiredLib) String() string {
	return fmt.Sprintf("name => %s, version => %s, dir => %s, wanted => %s",
		r.Name, r.Version, r.IncDir, r.Wanted)
}

func (r *RequiredLib) Download() error {
	log.Infof("Downloading: %s", r.String())

	body, err := GETRequestJSON(fmt.Sprintf("/v1/%s/lib/%s/versions/%s/files/%s/%s", depmanNs, r.Name, r.Version, depmanPlatform, depmanArch))

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
		log.Infof("  Downloading file: %s %s", file.Type, file.Name)

		var mode os.FileMode
		var dir string
		switch {
		case file.Type == "header" && strings.Contains(r.Wanted, "header"):
			dir = includeDir
			if r.IncDir != "" {
				dir = dir + r.IncDir
			}
			mode = 0644
		case file.Type == "archive" && strings.Contains(r.Wanted, "archive"):
			r.HasLib = true
			dir = libDir
			mode = 0644
		case file.Type == "object" && strings.Contains(r.Wanted, "object"):
			r.HasLib = true
			dir = libDir
			mode = 0644
		case file.Type == "shared" && strings.Contains(r.Wanted, "shared"):
			dir = libDir
			r.HasLib = true
			mode = 0755
		default:
			log.Warnf("Ignoring file %s of type %s as it is not wanted", file.Name, file.Type)
			continue
		}

		localfile, err := downloadLibFile(r.Name, r.Version, file, dir, mode)

		if err != nil {
			return err
		}

		// We downloaded a new header file - remember it
		if file.Type == "header" {
			new_header_files[localfile] = 1
		}

		for _, link := range file.Links {
			symlink := dir + "/" + link.Name
			log.Infof("    Symlink: %s => %s", symlink, localfile)

			os.Remove(symlink)
			err = os.Symlink(file.Name, symlink)
			if err != nil {
				return err
			}
		}
	}

	r.Downloaded = true

	return nil
}

type RequiredLibs struct {
	Libs []*RequiredLib
}

func (r *RequiredLibs) Has(l *RequiredLib) bool {
	for _, entry := range r.Libs {
		if entry.Name == l.Name && entry.Version == l.Version && entry.IncDir == l.IncDir && entry.Wanted == l.Wanted {
			return true
		}
	}
	return false
}

func (r *RequiredLibs) Add(l *RequiredLib) {
	if !r.Has(l) {
		r.Libs = append(r.Libs, l)
	}
}

func (r *RequiredLibs) Download() ([]string, error) {
	libnames := make([]string, 0)
	for _, lib := range r.Libs {
		if !lib.Downloaded {
			err := lib.Download()
			if err != nil {
				return libnames, err
			}
			if lib.HasLib {
				libnames = append(libnames, strings.TrimPrefix(lib.Name, "lib"))
			}
		} else {
			log.Debugf(" Already downloaded: %s", lib.String())
		}
	}

	return libnames, nil
}

func parseDepLine(line string) (*RequiredLib, error) {
	// split by pipe, surrounded by optional white space
	white := regexp.MustCompile("\\s*\\|\\s*")
	fields := white.Split(line, -1)

	lib := &RequiredLib{}
	lib.Wanted = "header,archive,shared"

	for idx, field := range fields {
		switch {
		case idx == 0:
			parts := strings.Split(field, ":")
			lib.Name = parts[0]
			if len(parts) > 1 {
				lib.Version = parts[1]
			} else {
				lib.Version = "latest"
			}
		case idx == 1:
			lib.IncDir = strings.TrimSuffix(field, "/")
			if !strings.HasPrefix(lib.IncDir, "/") {
				lib.IncDir = "/" + lib.IncDir
			}
		case idx == 2:
			lib.Wanted = field
		}
	}

	log.Debugf("Dependency: %s", lib.String())

	return lib, nil
}

func ParseDepfile(depfile string) (*RequiredLibs, error) {
	deps := &RequiredLibs{}

	fh, err := os.Open(depfile)
	if err != nil {
		return deps, errors.New(fmt.Sprintf("Cannot open dependency file %s: %s", depfile, err))
	}

	defer fh.Close()

	r := bufio.NewReader(fh)

	for err == nil {
		line, _, err := r.ReadLine()
		if err != nil && err != io.EOF {
			return deps, err
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
		deps.Add(dep)

		log.Infof("Want lib: %s", dep.String())
	}

	return deps, nil
}

func ParseSourceFiles(walkpath string, wanted string, deps *RequiredLibs) error {
	// First, find all #includes (de-duped) in source files
	found_hash_includes = make(map[string]string)
	err := filepath.Walk(walkpath, HashIncludeScanner)
	if err != nil {
		return err
	}

	// Now, see if they exist and add them to deps
	for include, sourcefile := range found_hash_includes {
		log.Infof("Found #include: %s in file %s", include, sourcefile)

		_, err := os.Stat(include)
		if err == nil {
			log.Infof("  File %s exists in local dir - skipping", include)
			continue
		}
		err = nil

		// Split path - file
		parts := strings.Split(include, "/")
		include_file := parts[len(parts)-1]
		parts = parts[:len(parts)-1]
		include_path := strings.Join(parts, "/")
		log.Debugf("  Wants include file '%s' in path '%s'", include_file, include_path)

		// Checking if the server has this header file that's being included
		library, version, err := findLibFromFile(include_file)
		if err != nil {
			log.Infof("  Cannot find library for file %s: %s", include_file, err)
			continue
		}
		log.Infof("  Header file %s provided by %s:%s", include_file, library, version)

		// Make req obj
		dep := &RequiredLib{}
		dep.Name = library
		dep.Version = version
		dep.IncDir = "/" + include_path
		if wanted != "" {
			dep.Wanted = wanted
		} else {
			dep.Wanted = "header,archive,shared"
		}

		// Add to global download list
		deps.Add(dep)
	}
	return nil
}

func HashIncludeScanner(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Errorf("Scanning error: %s", err)
		return err
	}

	log.Debugf("Scanning path: %s - dir %t", path, info.IsDir())

	if !info.IsDir() && info.Mode().IsRegular() {
		log.Debugf("  inspecting regular file: %s", path)
		includes, err := findHashIncludes(path)
		if err != nil {
			return err
		}

		for _, i := range includes {
			// Using string map here for free de-duping
			found_hash_includes[i] = path
		}
	}
	return nil
}

func findHashIncludes(filepath string) ([]string, error) {
	includes := make([]string, 0)
	fh, err := os.Open(filepath)
	if err != nil {
		return includes, err
	}
	defer fh.Close()

	r := bufio.NewReader(fh)

	// #include <framework.h>
	// # include "helpers.h"
	// #include <gtest/gtest.h>
	include_re := regexp.MustCompile("^#\\s*include\\s+[<\"](\\S+)[>\"]")

	for err == nil {
		line, _, err := r.ReadLine()
		if err != nil && err != io.EOF {
			log.Errorf("Got a readline error: %s", err)
			return includes, err
		} else if err == io.EOF {
			// Finished reading file
			break
		}

		matches := include_re.FindAllStringSubmatch(string(line), -1)

		if len(matches) > 0 {
			includes = append(includes, matches[0][1])
		}
	}

	return includes, nil
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
	fmt.Printf("DEPMAN_LIBS = -l%s\n", strings.Join(dedupeStringSlice(libnames), " \\\n\t-l"))
	fmt.Printf("DEPMAN_CFLAGS = -I$(DEPMAN_INC_DIR)\n")
	fmt.Printf("DEPMAN_CCLDFLAGS = -L$(DEPMAN_LIB_DIR) $(DEPMAN_LIBS)")
	fmt.Printf("\n")
}

// https://play.golang.org/p/q3bZ3hpOzD
func dedupeStringSlice(slice []string) []string {
	m := make(map[string]bool)

	for _, v := range slice {
		if _, seen := m[v]; !seen {
			slice[len(m)] = v
			m[v] = true
		}
	}

	slice = slice[:len(m)]

	return slice
}

func findLibFromFile(filename string) (string, string, error) {
	body, err := GETRequestJSON(fmt.Sprintf("/v1/%s/search/%s/%s/%s", depmanNs, depmanPlatform, depmanArch, filename))

	if err != nil {
		return "", "", err
	}
	files := depman.Files{}

	if err = json.Unmarshal(body, &files); err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	if len(files) > 1 {
		return "", "", fmt.Errorf("Found more than one file returned when searching for %s (have %d)", filename, len(files))
	}

	return files[0].Library, files[0].Version, nil
}

func downloadLibFile(libname string, libver string, f depman.File, dir string, mode os.FileMode) (string, error) {
	localfile := fmt.Sprintf("%s/%s", dir, f.Name)
	log.Debugf("Downloading %s/%s/%s to %s", libname, libver, f.Name, localfile)

	uri_path := fmt.Sprintf("/v1/%s/lib/%s/versions/%s/files/%s/%s/%s/%s/download",
		depmanNs, libname, libver, depmanPlatform, depmanArch, f.Type, f.Name)

	return localfile, doDownload(uri_path, localfile, mode)
}

func doDownload(url string, localfile string, mode os.FileMode) error {
	log.Debugf("Downloading from %s to %s", url, localfile)
	filedir := filepath.Dir(localfile)
	filename := filepath.Base(localfile)

	tmpfile := fmt.Sprintf("%s/.%s.dwn", filedir, filename)
	if _, err := os.Stat(filedir); os.IsNotExist(err) {
		err = os.MkdirAll(filedir, 0755)
		if err != nil {
			return err
		}
	}

	for _, fname := range []string{tmpfile, localfile} {
		_, err := os.Stat(fname)
		if err == nil {
			log.Debugf("File %s exists - removing", fname)
			os.Remove(fname)
		}
	}

	fh, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE, mode)
	defer fh.Close()
	if err != nil {
		return err
	}

	err = downloadToFh(url, fh)
	if err != nil {
		fh.Close()
		os.Remove(tmpfile)
		return err
	}

	fh.Close()

	err = os.Rename(tmpfile, localfile)

	return err
}

func downloadToFh(url string, fh io.Writer) error {
	log.Debugf("Download URL: %s", url)

	req_url := strings.Join([]string{depmanUrl, url}, "")

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

	written, err := io.Copy(fh, resp.Body)
	if err != nil {
		return err
	}
	log.Debugf("Wrote %d bytes", written)

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
