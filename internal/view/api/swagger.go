package api

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/labstack/echo/v4"
)

//go:embed openapi.yaml
var openAPISpecTemplate []byte

const serverURLPlaceholder = "SERVER_URL_PLACEHOLDER"

func (h *handlers) openAPIHandler(c echo.Context) error {
	spec := strings.ReplaceAll(
		string(openAPISpecTemplate),
		serverURLPlaceholder,
		pathutil.BuildPath("/api/v1"),
	)
	return c.Blob(http.StatusOK, "application/yaml", []byte(spec))
}

func (h *handlers) swaggerUIHandler(c echo.Context) error {
	specURL := pathutil.BuildPath("/api/v1/openapi.yaml")
	html := `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>PG Warden API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({
        url: "` + specURL + `",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "StandaloneLayout"
      });
    };
  </script>
</body>
</html>`
	return c.HTML(http.StatusOK, html)
}
