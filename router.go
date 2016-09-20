package depman

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

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
		handler = route.HandlerFunc
		//handler = c.ClientHandler(handler, route.Name)
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
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
			"ListLibraries",
			"GET",
			"/v1/lib/{ns}",
			HandleListLibraries,
		},
		Route{
			"GetLibrary",
			"GET",
			"/v1/lib/{ns}/{libname}",
			HandleGetLibrary,
		},
		Route{
			"GetLibrary",
			"GET",
			"/v1/lib/{ns}/{libname}/versions",
			HandleGetLibraryVersions,
		},
		Route{
			"GetLibrary",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}",
			HandleGetLibraryVersion,
		},
		Route{
			"GetLibraryFiles",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files",
			HandleGetLibraryFiles,
		},
		Route{
			"GetLibraryFilesPlatform",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}",
			HandleGetLibraryFilesPlatform,
		},
		Route{
			"GetLibraryFilesPlatformArch",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}",
			HandleGetLibraryFilesPlatformArch,
		},
		Route{
			"GetLibraryFilesPlatformArchType",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}",
			HandleGetLibraryFilesPlatformArchType,
		},
		Route{
			"GetLibraryFilesPlatformArchType",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}",
			HandleGetLibraryFilesPlatformArchType,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeName",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}",
			HandleGetLibraryFilesPlatformArchTypeName,
		},
		Route{
			"PutLibraryFilesPlatformArchTypeName",
			"PUT",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}",
			HandlePutLibraryFilesPlatformArchTypeName,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeNameLinks",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}/links",
			HandleGetLibraryFilesPlatformArchTypeNameLinks,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeNameLinks",
			"PUT",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}/links/{linkname}",
			HandleAddLink,
		},
		Route{
			"FileDownload",
			"GET",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}/download",
			HandleFileDownload,
		},
		Route{
			"FileUpload",
			"PUT",
			"/v1/lib/{ns}/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}/upload",
			HandleFileUpload,
		},
	}
}
