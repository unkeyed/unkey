package oapicodegen_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

func TestYAMLPathPreservesOpenAPIValues(t *testing.T) {
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.1.0
paths:
  /users/{id}:
    get:
      tags: ["0123", "true"]
      responses:
        "200":
          description: ok
`), &document))
	path, err := yamlpath.NewPath(`$.paths['/users/{id}'].get.tags[*]`)
	require.NoError(t, err)
	nodes, err := path.Find(&document)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	require.Equal(t, "0123", nodes[0].Value)
	require.Equal(t, "true", nodes[1].Value)
	for _, node := range nodes {
		require.Equal(t, "!!str", node.Tag)
	}

	path, err = yamlpath.NewPath(`$.paths['/users/{id}'].get.responses['200'].description`)
	require.NoError(t, err)
	nodes, err = path.Find(&document)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "ok", nodes[0].Value)
}
