package main

import (
	"context"
	"path/filepath"
	"testing"

	pluginpb "github.com/infracost/proto/gen/go/infracost/plugin"
)

func fixtures(t *testing.T, parts ...string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join(append([]string{"..", "..", "testdata", "fixtures"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestGetPluginInfo(t *testing.T) {
	resp, err := (&service{}).GetPluginInfo(context.Background(), &pluginpb.GetPluginInfoRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetName() != "infracost/expectedload" || resp.GetType() != pluginpb.PluginType_PARSER {
		t.Fatalf("unexpected plugin info: %+v", resp)
	}
	if resp.GetVersion() != version {
		t.Errorf("version = %q, want %q", resp.GetVersion(), version)
	}
}

func TestGetParserConfig(t *testing.T) {
	resp, err := (&service{}).GetParserConfig(context.Background(), &pluginpb.GetParserConfigRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetIdentificationPriority() != 45 || resp.GetConfigFileProjectType() != "expectedload" {
		t.Fatalf("unexpected parser config: %+v", resp)
	}
}

func TestIdentifyProjects(t *testing.T) {
	svc := &service{}
	for name, tc := range map[string]struct {
		path string
		want bool
	}{
		"empty":             {},
		"missing":           {path: filepath.Join(t.TempDir(), "missing")},
		"with declarations": {path: fixtures(t, "terraform"), want: true},
	} {
		t.Run(name, func(t *testing.T) {
			resp, err := svc.IdentifyProjects(context.Background(), &pluginpb.IdentifyProjectsRequest{Directory: tc.path})
			if err != nil {
				t.Fatal(err)
			}
			if resp.GetDirectory() != tc.want {
				t.Errorf("directory = %v, want %v", resp.GetDirectory(), tc.want)
			}
		})
	}
}

func TestParseRequiresPath(t *testing.T) {
	if _, err := (&service{}).Parse(context.Background(), &pluginpb.ParseRequest{}); err == nil {
		t.Fatal("Parse without path should fail")
	}
}

func TestParseReturnsTreeAndDiagnostics(t *testing.T) {
	resp, err := (&service{}).Parse(context.Background(), &pluginpb.ParseRequest{Path: fixtures(t)})
	if err != nil {
		t.Fatal(err)
	}
	svc := resp.GetTree().GetProviders()["expectedload"].GetServices()["expected-load"]
	if svc == nil {
		t.Fatal("tree missing expectedload provider/service")
	}
	if len(svc.GetResources()) != 11 {
		t.Fatalf("got %d tree resources, want 11", len(svc.GetResources()))
	}
	if len(resp.GetDiagnostics()) != 3 {
		t.Fatalf("got %d diagnostics, want 3", len(resp.GetDiagnostics()))
	}
	for _, r := range svc.GetResources() {
		if r.GetDefinition().GetSource().GetFilename() == "terraform/main.tf" && r.GetDefinition().GetSource().GetStartLine() == 1 {
			if got := r.GetAttributes().GetEntries()["monthly_requests"].GetIntValue(); got != 5_000_000 {
				t.Errorf("monthly_requests = %d, want 5000000", got)
			}
			return
		}
	}
	t.Fatal("terraform/main.tf:1 resource missing")
}

func TestParseSingleFile(t *testing.T) {
	resp, err := (&service{}).Parse(context.Background(), &pluginpb.ParseRequest{Path: fixtures(t, "rust", "lib.rs")})
	if err != nil {
		t.Fatal(err)
	}
	resources := resp.GetTree().GetProviders()["expectedload"].GetServices()["expected-load"].GetResources()
	if len(resources) != 1 || resources[0].GetDefinition().GetSource().GetFilename() != "lib.rs" {
		t.Fatalf("unexpected resources: %+v", resources)
	}
}
