package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	_ "shadow-dom/etlctl/pkg/util/etl/sources"
	_ "shadow-dom/etlctl/pkg/util/etl/targets"
	"shadow-dom/etlctl/web/server/handler"
	"shadow-dom/etlctl/web/server/middleware"
	"shadow-dom/etlctl/web/server/service"
)

func main() {
	port := flag.Int("port", 8080, "Server port")
	configDir := flag.String("config-dir", "./etls", "ETL config directory")
	uiDir := flag.String("ui-dir", "", "Path to built UI static files (optional)")
	flag.Parse()

	// Services
	etlSvc := service.NewETLService(*configDir)
	runnerSvc := service.NewRunnerService(*configDir)
	dagSvc := service.NewDAGService(*configDir)

	// Handlers
	etlH := handler.NewETLHandler(etlSvc)
	runH := handler.NewRunHandler(runnerSvc)
	dagH := handler.NewDAGHandler(dagSvc)
	regH := handler.NewRegistryHandler()
	schemaH := handler.NewSchemaHandler()

	mux := http.NewServeMux()

	// ETL CRUD
	mux.HandleFunc("GET /api/v1/etls", etlH.List)
	mux.HandleFunc("POST /api/v1/etls", etlH.Create)
	mux.HandleFunc("GET /api/v1/etls/{name}", etlH.Get)
	mux.HandleFunc("PUT /api/v1/etls/{name}", etlH.Update)
	mux.HandleFunc("DELETE /api/v1/etls/{name}", etlH.Delete)

	// Validation, run, test
	mux.HandleFunc("POST /api/v1/etls/{name}/validate", etlH.Validate)
	mux.HandleFunc("POST /api/v1/etls/{name}/run", runH.RunETL)
	mux.HandleFunc("POST /api/v1/etls/{name}/test", runH.TestETL)

	// DAG metadata
	mux.HandleFunc("GET /api/v1/etls/{name}/dag", dagH.Get)
	mux.HandleFunc("PUT /api/v1/etls/{name}/dag", dagH.Save)

	// Registry
	mux.HandleFunc("GET /api/v1/registry/sources", regH.SourceTypes)
	mux.HandleFunc("GET /api/v1/registry/targets", regH.TargetTypes)
	mux.HandleFunc("GET /api/v1/registry/transforms", regH.Transforms)
	mux.HandleFunc("POST /api/v1/transforms/preview", regH.PreviewTransform)

	// Schema introspection
	mux.HandleFunc("POST /api/v1/schema/introspect", schemaH.Introspect)
	mux.HandleFunc("POST /api/v1/schema/describe", schemaH.Describe)
	mux.HandleFunc("POST /api/v1/schema/auto-map", schemaH.AutoMap)

	// Serve UI static files if provided
	if *uiDir != "" {
		uiFS := os.DirFS(*uiDir)
		fileServer := http.FileServerFS(uiFS)
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			// Try to serve the file; fall back to index.html for SPA routing
			path := r.URL.Path
			if path == "/" {
				path = "index.html"
			}
			if _, err := fs.Stat(uiFS, path[1:]); err != nil {
				// SPA fallback
				http.ServeFileFS(w, r, uiFS, "index.html")
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("etlctl server starting on %s (config-dir: %s)\n", addr, *configDir)
	if err := http.ListenAndServe(addr, middleware.CORS(mux)); err != nil {
		log.Fatal(err)
	}
}
