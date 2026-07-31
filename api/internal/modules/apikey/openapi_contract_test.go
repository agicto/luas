package apikey

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type openAPIContract struct {
	Components struct {
		Schemas map[string]struct {
			Properties map[string]yaml.Node `yaml:"properties"`
		} `yaml:"schemas"`
	} `yaml:"components"`
}

func TestAPIKeyDTOFieldsMatchOpenAPI(t *testing.T) {
	contractPath := filepath.Join("..", "..", "..", "..", "contracts", "openapi.yaml")
	content, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}

	var contract openAPIContract
	if err := yaml.Unmarshal(content, &contract); err != nil {
		t.Fatalf("parse OpenAPI contract: %v", err)
	}

	assertOpenAPIProperties(t, contract, "APIKey", APIKeyResponse{})
	assertOpenAPIProperties(t, contract, "CreateAPIKeyRequest", APIKeyCreateRequest{})
	assertOpenAPIProperties(t, contract, "CreateAPIKeyResult", APIKeyCreateResponse{})
}

func assertOpenAPIProperties(t *testing.T, contract openAPIContract, schemaName string, dto any) {
	t.Helper()
	schema, exists := contract.Components.Schemas[schemaName]
	if !exists {
		t.Fatalf("OpenAPI schema %q is missing", schemaName)
	}

	schemaFields := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		schemaFields = append(schemaFields, name)
	}
	sort.Strings(schemaFields)

	typeOfDTO := reflect.TypeOf(dto)
	dtoFields := make([]string, 0, typeOfDTO.NumField())
	for index := 0; index < typeOfDTO.NumField(); index++ {
		name := strings.Split(typeOfDTO.Field(index).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			dtoFields = append(dtoFields, name)
		}
	}
	sort.Strings(dtoFields)

	if !reflect.DeepEqual(dtoFields, schemaFields) {
		t.Fatalf("%s DTO fields = %v, OpenAPI properties = %v", schemaName, dtoFields, schemaFields)
	}
}
