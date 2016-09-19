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
			"/lib",
			HandleListLibraries,
		},
		Route{
			"GetLibrary",
			"GET",
			"/lib/{libname}",
			HandleGetLibrary,
		},
		Route{
			"GetLibrary",
			"GET",
			"/lib/{libname}/versions",
			HandleGetLibraryVersions,
		},
		Route{
			"GetLibrary",
			"GET",
			"/lib/{libname}/versions/{libver}",
			HandleGetLibraryVersion,
		},
		Route{
			"GetLibraryFiles",
			"GET",
			"/lib/{libname}/versions/{libver}/files",
			HandleGetLibraryFiles,
		},
		Route{
			"GetLibraryFilesPlatform",
			"GET",
			"/lib/{libname}/versions/{libver}/files/{platform}",
			HandleGetLibraryFilesPlatform,
		},
		Route{
			"GetLibraryFilesPlatformArch",
			"GET",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}",
			HandleGetLibraryFilesPlatformArch,
		},
		Route{
			"GetLibraryFilesPlatformArchType",
			"GET",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}",
			HandleGetLibraryFilesPlatformArchType,
		},
		Route{
			"GetLibraryFilesPlatformArchType",
			"GET",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}",
			HandleGetLibraryFilesPlatformArchType,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeName",
			"GET",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}",
			HandleGetLibraryFilesPlatformArchTypeName,
		},
		Route{
			"PutLibraryFilesPlatformArchTypeName",
			"PUT",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}",
			HandlePutLibraryFilesPlatformArchTypeName,
		},
		Route{
			"GetLibraryFilesPlatformArchTypeNameLinks",
			"GET",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}/links",
			HandleGetLibraryFilesPlatformArchTypeNameLinks,
		},
		Route{
			"FileDownload",
			"GET",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}/download",
			HandleFileDownload,
		},
		Route{
			"FileUpload",
			"PUT",
			"/lib/{libname}/versions/{libver}/files/{platform}/{arch}/{filetype}/{filename}/upload",
			HandleFileUpload,
		},
	}
}
