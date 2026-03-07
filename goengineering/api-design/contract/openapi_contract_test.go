package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenAPISpecLintAndRequiredContracts(t *testing.T) {
	spec := loadSpec(t, specPath(t, "../openapi/openapi.json"))

	openapiVersion := stringValue(spec, "openapi")
	if !strings.HasPrefix(openapiVersion, "3.") {
		t.Fatalf("openapi must be v3, got %q", openapiVersion)
	}

	required := map[string][]string{
		"/healthz":           {"get"},
		"/api/v1/users":      {"get", "post"},
		"/api/v1/users/{id}": {"get", "put", "patch", "delete"},
		"/api/v2/users":      {"get"},
		"/api/v2/users/{id}": {"get"},
	}
	paths := mapValue(spec, "paths")
	for path, methods := range required {
		pathItem, ok := paths[path].(map[string]any)
		if !ok {
			t.Fatalf("missing path: %s", path)
		}
		for _, method := range methods {
			if _, ok := pathItem[method]; !ok {
				t.Fatalf("missing method %s on path %s", method, path)
			}
		}
	}

	v1User := mustPathMethod(t, paths, "/api/v1/users/{id}", "put")
	assertHasRequiredIfMatch(t, v1User)
	patch := mustPathMethod(t, paths, "/api/v1/users/{id}", "patch")
	assertHasRequiredIfMatch(t, patch)

	errorBody := mapValue(spec, "components", "schemas", "ErrorBody", "properties")
	for _, field := range []string{"trace_id", "metric", "audit"} {
		if _, ok := errorBody[field]; !ok {
			t.Fatalf("ErrorBody missing observability field %q", field)
		}
	}
}

func TestOpenAPIBreakingChangeGuard(t *testing.T) {
	current := loadSpec(t, specPath(t, "../openapi/openapi.json"))
	baseline := loadSpec(t, specPath(t, "../openapi/baseline/openapi.v1.json"))

	currentPaths := mapValue(current, "paths")
	baselinePaths := mapValue(baseline, "paths")

	for path, baselinePathRaw := range baselinePaths {
		baselinePath, ok := baselinePathRaw.(map[string]any)
		if !ok {
			t.Fatalf("baseline path %s malformed", path)
		}
		currentPathRaw, ok := currentPaths[path]
		if !ok {
			t.Fatalf("breaking change: removed path %s", path)
		}
		currentPath, ok := currentPathRaw.(map[string]any)
		if !ok {
			t.Fatalf("current path %s malformed", path)
		}

		for method, baselineOpRaw := range baselinePath {
			if !isHTTPMethod(method) {
				continue
			}
			baselineOp, ok := baselineOpRaw.(map[string]any)
			if !ok {
				t.Fatalf("baseline operation %s %s malformed", strings.ToUpper(method), path)
			}
			currentOpRaw, ok := currentPath[method]
			if !ok {
				t.Fatalf("breaking change: removed operation %s %s", strings.ToUpper(method), path)
			}
			currentOp, ok := currentOpRaw.(map[string]any)
			if !ok {
				t.Fatalf("current operation %s %s malformed", strings.ToUpper(method), path)
			}

			baselineResp := mapFromOpResponses(baselineOp)
			currentResp := mapFromOpResponses(currentOp)
			for code := range baselineResp {
				if _, ok := currentResp[code]; !ok {
					t.Fatalf("breaking change: removed response code %s for %s %s", code, strings.ToUpper(method), path)
				}
			}
		}
	}
}

func assertHasRequiredIfMatch(t *testing.T, op map[string]any) {
	t.Helper()
	params, ok := op["parameters"].([]any)
	if !ok {
		t.Fatal("operation missing parameters")
	}
	for _, p := range params {
		paramMap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		ref, _ := paramMap["$ref"].(string)
		if ref == "#/components/parameters/IfMatch" {
			return
		}
		if strings.EqualFold(stringValue(paramMap, "name"), "If-Match") && boolValue(paramMap, "required") {
			return
		}
	}
	t.Fatal("operation missing required If-Match parameter")
}

func mustPathMethod(t *testing.T, paths map[string]any, path, method string) map[string]any {
	t.Helper()
	pathItem, ok := paths[path].(map[string]any)
	if !ok {
		t.Fatalf("path %s not found", path)
	}
	op, ok := pathItem[method].(map[string]any)
	if !ok {
		t.Fatalf("method %s on path %s not found", method, path)
	}
	return op
}

func mapFromOpResponses(op map[string]any) map[string]any {
	if op == nil {
		return map[string]any{}
	}
	responses, ok := op["responses"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return responses
}

func isHTTPMethod(v string) bool {
	switch strings.ToLower(v) {
	case "get", "post", "put", "patch", "delete", "head", "options", "trace":
		return true
	default:
		return false
	}
}

func loadSpec(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spec %s: %v", path, err)
	}
	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse spec %s: %v", path, err)
	}
	if len(spec) == 0 {
		t.Fatalf("spec %s is empty", path)
	}
	return spec
}

func specPath(t *testing.T, relative string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve runtime caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), relative))
}

func mapValue(root map[string]any, keys ...string) map[string]any {
	current := any(root)
	for _, key := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return map[string]any{}
		}
		next, ok := asMap[key]
		if !ok {
			return map[string]any{}
		}
		current = next
	}
	asMap, ok := current.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return asMap
}

func stringValue(root map[string]any, keys ...string) string {
	current := any(root)
	for _, key := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		next, ok := asMap[key]
		if !ok {
			return ""
		}
		current = next
	}
	value, _ := current.(string)
	return value
}

func boolValue(root map[string]any, keys ...string) bool {
	current := any(root)
	for _, key := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return false
		}
		next, ok := asMap[key]
		if !ok {
			return false
		}
		current = next
	}
	value, _ := current.(bool)
	return value
}
