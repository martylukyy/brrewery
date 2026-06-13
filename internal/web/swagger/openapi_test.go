package swagger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOpenAPISpec(t *testing.T) {
	require.NotEmpty(t, openapiYAML)

	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(openapiYAML, &spec))

	require.NotNil(t, spec["openapi"])
	require.NotNil(t, spec["info"])
	require.NotNil(t, spec["paths"])

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok)

	endpoints := 0
	for _, pathItem := range paths {
		methods, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}
		for method := range methods {
			switch method {
			case "get", "post", "put", "delete", "patch":
				endpoints++
			}
		}
	}

	require.GreaterOrEqual(t, endpoints, 6)

	components, ok := spec["components"].(map[string]any)
	require.True(t, ok)
	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok)

	for _, name := range []string{"LoginRequest", "App", "AppListResponse", "ErrorResponse"} {
		require.NotNil(t, schemas[name], "missing schema %s", name)
	}
}

func TestOpenAPISecuritySchemes(t *testing.T) {
	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(openapiYAML, &spec))

	components := spec["components"].(map[string]any)
	schemes := components["securitySchemes"].(map[string]any)
	require.NotNil(t, schemes["SessionAuth"])
}
