package cmd

import (
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

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI and API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		uiDir, _ := cmd.Flags().GetString("ui-dir")

		etlSvc := service.NewETLService(configDir)
		runnerSvc := service.NewRunnerService(configDir)
		dagSvc := service.NewDAGService(configDir)
		connSvc := service.NewConnectionService(configDir)
		versionSvc := service.NewVersionService(configDir)

		etlH := handler.NewETLHandler(etlSvc)
		runH := handler.NewRunHandler(runnerSvc)
		dagH := handler.NewDAGHandler(dagSvc)
		regH := handler.NewRegistryHandler()
		schemaH := handler.NewSchemaHandler()
		connH := handler.NewConnectionHandler(connSvc)
		versionH := handler.NewVersionHandler(versionSvc)

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
		mux.HandleFunc("GET /api/v1/registry/functions", regH.RegisteredFunctions)
		mux.HandleFunc("POST /api/v1/transforms/preview", regH.PreviewTransform)

		// Raw YAML editing
		mux.HandleFunc("GET /api/v1/etls/{name}/yaml", etlH.GetRawYAML)
		mux.HandleFunc("PUT /api/v1/etls/{name}/yaml", etlH.UpdateRawYAML)

		// Schema introspection & auto-map
		mux.HandleFunc("POST /api/v1/schema/introspect", schemaH.Introspect)
		mux.HandleFunc("POST /api/v1/schema/describe", schemaH.Describe)
		mux.HandleFunc("POST /api/v1/schema/auto-map", schemaH.AutoMap)

		// Connections
		mux.HandleFunc("GET /api/v1/connections", connH.List)
		mux.HandleFunc("GET /api/v1/connections/{name}", connH.Get)
		mux.HandleFunc("POST /api/v1/connections", connH.Save)
		mux.HandleFunc("PUT /api/v1/connections/{name}", connH.Save)
		mux.HandleFunc("DELETE /api/v1/connections/{name}", connH.Delete)

		// Versions
		mux.HandleFunc("GET /api/v1/etls/{name}/versions", versionH.List)
		mux.HandleFunc("POST /api/v1/etls/{name}/versions", versionH.Save)
		mux.HandleFunc("POST /api/v1/etls/{name}/versions/{version}/restore", versionH.Restore)

		// Serve UI static files
		if uiDir != "" {
			uiFS := os.DirFS(uiDir)
			fileServer := http.FileServerFS(uiFS)
			mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if path == "/" {
					path = "index.html"
				}
				if _, err := fs.Stat(uiFS, path[1:]); err != nil {
					http.ServeFileFS(w, r, uiFS, "index.html")
					return
				}
				fileServer.ServeHTTP(w, r)
			})
		}

		addr := fmt.Sprintf(":%d", port)
		log.Printf("etlctl server starting on http://localhost%s (config-dir: %s)\n", addr, configDir)
		return http.ListenAndServe(addr, middleware.CORS(mux))
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Server port")
	serveCmd.Flags().String("ui-dir", "", "Path to built UI static files")
	rootCmd.AddCommand(serveCmd)
}
