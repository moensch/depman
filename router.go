package depman

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

const RequestNS = 0

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	for _, route := range RouteDefinitions() {
		log.Debugf("Setting up route %s for %s %s", route.Name, route.Method, route.Pattern)
		var handler http.Handler
		handler = handlerDecorate(route.HandlerFunc)
		//handler = c.ClientHandler(handler, route.Name)
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

func handlerDecorate(f http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqVars := mux.Vars(r)
		if _, ok := reqVars["ns"]; ok {
			// Have namespace - remember for the context of this request
			context.Set(r, RequestNS, reqVars["ns"])
		}

		f(w, r)
		log.WithFields(log.Fields{
			"method": r.Method,
			"uri":    r.RequestURI,
			"client": r.RemoteAddr,
			"time":   time.Since(start),
		}).Info("Request")
		context.Clear(r)
	})
}

func RouteDefinitions() Routes {
	return Routes{
		Route{
			"Index",
			"GET",
			"/",
			HandleIndex,
		},
		Route{
			"FindFile",
			"GET",
			"/v1/{ns}/search/{platform}/{arch}/{name}",
			HandleListFiles,
		},
		Route{
			"ListLibraries",
			"GET",
			"/v1/{ns}/lib",
			HandleListLibraries,
		},
		Route{
			"GetLibrary",
			"GET",
			"/v1/{ns}/lib/{library}",
			HandleGetLibrary,
		},
		Route{
			"GetLibrary",
			"GET",
			"/v1/{ns}/lib/{library}/versions",
			HandleListLibraryVersions,
		},
		Route{
			"GetLibrary",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}",
			HandleGetLibraryVersion,
		},
		Route{
			"GetLibraryFiles",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}/files",
			HandleListFiles,
		},
		Route{
			"GetLibraryFilesPlatform",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}",
			HandleListFiles,
		},
		Route{
			"GetLibraryFilesPlatformArch",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}",
			HandleListFiles,
		},
		Route{
			"GetLibraryFilesPlatformArchType",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}/{type}",
			HandleListFiles,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeName",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}/{type}/{name}",
			HandleListFiles,
		},
		Route{
			"PutLibraryFilesPlatformArchTypeName",
			"PUT",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}/{type}/{name}",
			HandlePutFile,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeNameLinks",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}/{type}/{name}/links",
			HandleGetFileLinks,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeNameLinks",
			"PUT",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}/{type}/{name}/links/{linkname}",
			HandlePutLink,
		},
		Route{
			"FileDownload",
			"GET",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}/{type}/{name}/download",
			HandleFileDownload,
		},
		Route{
			"FileUpload",
			"PUT",
			"/v1/{ns}/lib/{library}/versions/{version}/files/{platform}/{arch}/{type}/{name}/upload",
			HandleFileUpload,
		},
	}
}
