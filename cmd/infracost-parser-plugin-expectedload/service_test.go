package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/infracost/expectedload/internal/scan"
	"github.com/infracost/proto/gen/go/infracost/parser/api"
	"github.com/infracost/proto/gen/go/infracost/parser/cloudformation"
	"github.com/infracost/proto/gen/go/infracost/parser/terraform"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func fixtures(t *testing.T, parts ...string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join(append([]string{"..", "..", "testdata", "fixtures"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestDescribe(t *testing.T) {
	resp, err := (&service{}).Describe(context.Background(), &api.DescribeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetName() != "plugins.infracost.io/infracost/expectedload" {
		t.Errorf("name = %q", resp.GetName())
	}
	if resp.GetDisplayName() != "Expected Load" {
		t.Errorf("display name = %q", resp.GetDisplayName())
	}
	if resp.GetPriority() != 45 {
		t.Errorf("priority = %d, want 45", resp.GetPriority())
	}
	if len(resp.GetFileExtensions()) != 10 {
		t.Errorf("extensions = %v, want 10 entries", resp.GetFileExtensions())
	}
	if !resp.GetSupportsDirectories() {
		t.Error("supports_directories = false, want true")
	}
}

func TestDetectRPC(t *testing.T) {
	svc := &service{}
	ctx := context.Background()

	for name, req := range map[string]*api.DetectRequest{
		"nil-ish empty":      {},
		"empty path":         {Path: ""},
		"nonexistent":        {Path: filepath.Join(t.TempDir(), "nope.tf")},
		"unsupported ext":    {Path: "whatever.yaml"},
		"content, no marker": {Path: "a.go", Content: []byte("package a\n"), ContentProvided: true},
	} {
		resp, err := svc.Detect(ctx, req)
		if err != nil {
			t.Errorf("%s: Detect errored: %v", name, err)
			continue
		}
		if resp.GetDetected() {
			t.Errorf("%s: detected = true, want false", name)
		}
	}

	resp, err := svc.Detect(ctx, &api.DetectRequest{Path: fixtures(t, "terraform", "main.tf")})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetDetected() {
		t.Fatal("fixture file not detected")
	}
	if resp.GetProjectType() != "expectedload" {
		t.Errorf("project_type = %q", resp.GetProjectType())
	}
	if resp.GetConfidence() != api.DetectConfidence_DETECT_CONFIDENCE_MEDIUM {
		t.Errorf("confidence = %v, want MEDIUM", resp.GetConfidence())
	}
}

func TestDetectRPCContentProvidedNeverReadsDisk(t *testing.T) {
	// The path doesn't exist on disk; only the provided content matters.
	resp, err := (&service{}).Detect(context.Background(), &api.DetectRequest{
		Path:            "/nonexistent/virtual.ts",
		Content:         []byte("// expected-load: monthly_calls=1\n"),
		ContentProvided: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetDetected() {
		t.Error("virtual document content not detected")
	}
}

func TestInitialize(t *testing.T) {
	if _, err := (&service{}).Initialize(context.Background(), &api.InitializeRequest{}); err != nil {
		t.Fatalf("Initialize must not error: %v", err)
	}
}

func TestParseNilHandling(t *testing.T) {
	svc := &service{}
	ctx := context.Background()

	if _, err := svc.Parse(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Errorf("Parse(nil) error = %v, want InvalidArgument", err)
	}
	if _, err := svc.Parse(ctx, &api.ParseRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("Parse(no target) error = %v, want InvalidArgument", err)
	}
	if _, err := svc.ParseToTree(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Errorf("ParseToTree(nil) error = %v, want InvalidArgument", err)
	}
	if _, err := svc.ParseToTree(ctx, &api.ParseToTreeRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("ParseToTree(no target) error = %v, want InvalidArgument", err)
	}
}

func terraformTarget(dir string) *api.ParseRequestTarget {
	return &api.ParseRequestTarget{
		Value: &api.ParseRequestTarget_Terraform{Terraform: &terraform.Target{Directory: dir}},
	}
}

func TestParseTerraformVariant(t *testing.T) {
	root := fixtures(t)
	resp, err := (&service{}).Parse(context.Background(), &api.ParseRequest{
		RepoDirectory:    root,
		WorkingDirectory: filepath.Join(root, "terraform"),
		Target:           terraformTarget(""),
	})
	if err != nil {
		t.Fatal(err)
	}

	tf := resp.GetResult().GetTerraform()
	if tf == nil {
		t.Fatal("no terraform result variant")
	}
	if len(tf.GetResources()) != 2 {
		t.Fatalf("got %d resources, want 2", len(tf.GetResources()))
	}
	r := tf.GetResources()[0]
	if r.GetType() != "expected_load" {
		t.Errorf("resource type = %q", r.GetType())
	}
	// Declaration paths are relative to repo_directory.
	if got := r.GetSourceRange().GetFilename(); got != "terraform/main.tf" {
		t.Errorf("filename = %q, want terraform/main.tf", got)
	}
	if r.GetSourceRange().GetStartLine() != 1 {
		t.Errorf("start line = %d, want 1", r.GetSourceRange().GetStartLine())
	}
	if r.GetSupported() {
		t.Error("declarations must not claim supported=true")
	}

	entries := r.GetData().GetMap().GetEntries()
	byKey := map[string]string{}
	for _, e := range entries {
		p := e.GetValue().GetPrimitive()
		byKey[e.GetKey()] = p.GetNumberValue() + p.GetStringValue()
	}
	if byKey["monthly_requests"] != "5000000" {
		t.Errorf("monthly_requests = %q, want 5000000", byKey["monthly_requests"])
	}
	if byKey["confidence"] != "high" {
		t.Errorf("confidence = %q, want high", byKey["confidence"])
	}
	if byKey["version"] != "1" {
		t.Errorf("version = %q, want 1", byKey["version"])
	}
}

func TestParseCloudformationVariant(t *testing.T) {
	root := fixtures(t, "python")
	resp, err := (&service{}).Parse(context.Background(), &api.ParseRequest{
		RepoDirectory:    root,
		WorkingDirectory: root,
		Target: &api.ParseRequestTarget{
			Value: &api.ParseRequestTarget_Cloudformation{Cloudformation: &cloudformation.Target{}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	cf := resp.GetResult().GetCloudformation()
	if cf == nil {
		t.Fatal("no cloudformation result variant")
	}
	if len(cf.GetResources()) != 2 {
		t.Fatalf("got %d resources, want 2", len(cf.GetResources()))
	}
	for _, r := range cf.GetResources() {
		if r.GetMetadata()["monthly_calls"].GetScalar().GetIntValue() == 0 {
			t.Errorf("%s: monthly_calls metadata missing", r.GetId())
		}
	}
}

func TestParsePartialResultsWithDiagnostics(t *testing.T) {
	root := fixtures(t, "errors")
	resp, err := (&service{}).Parse(context.Background(), &api.ParseRequest{
		RepoDirectory:    root,
		WorkingDirectory: root,
		Target:           terraformTarget(""),
	})
	if err != nil {
		t.Fatalf("recoverable problems must not fail the RPC: %v", err)
	}
	if len(resp.GetResult().GetTerraform().GetResources()) != 1 {
		t.Error("partial results missing: want the one parseable declaration")
	}

	var warns, errs int
	for _, d := range resp.GetDiagnostics() {
		if d.GetSourceRange().GetFilename() != "bad.py" {
			t.Errorf("diagnostic file = %q, want bad.py", d.GetSourceRange().GetFilename())
		}
		if d.GetWarning() {
			warns++
		} else {
			errs++
		}
	}
	if errs != 1 || warns != 2 {
		t.Errorf("got %d errors, %d warnings; want 1, 2", errs, warns)
	}
}

func TestParseUnreadableFileIsDiagnosticNotError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission bits are not enforced")
	}
	tmp := t.TempDir()
	locked := filepath.Join(tmp, "locked.tf")
	if err := os.WriteFile(locked, []byte("# expected-load: storage_gb=1\n"), 0o000); err != nil {
		t.Fatal(err)
	}

	resp, err := (&service{}).Parse(context.Background(), &api.ParseRequest{
		RepoDirectory:    tmp,
		WorkingDirectory: tmp,
		Target:           terraformTarget(""),
	})
	if err != nil {
		t.Fatalf("unreadable file must not fail the RPC: %v", err)
	}
	if len(resp.GetDiagnostics()) != 1 {
		t.Fatalf("want one diagnostic, got %+v", resp.GetDiagnostics())
	}
}

func TestParseUnmappedTargetVariant(t *testing.T) {
	// Terragrunt exists in the target oneof but has no result variant: the
	// response carries diagnostics and no result rather than an error.
	resp, err := (&service{}).Parse(context.Background(), &api.ParseRequest{
		RepoDirectory:    fixtures(t),
		WorkingDirectory: fixtures(t),
		Target:           &api.ParseRequestTarget{},
	})
	if err != nil {
		t.Fatalf("unmapped target must be recoverable: %v", err)
	}
	if resp.GetResult() != nil {
		t.Error("unmapped target should produce an empty result")
	}
	if len(resp.GetDiagnostics()) == 0 {
		t.Error("unmapped target should carry an explanatory diagnostic")
	}
}

func TestParseToTree(t *testing.T) {
	root := fixtures(t)
	resp, err := (&service{}).ParseToTree(context.Background(), &api.ParseToTreeRequest{
		RepoDirectory:    root,
		WorkingDirectory: root,
		Target:           terraformTarget(""),
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := resp.GetTree().GetProviders()["expectedload"].GetServices()["expected-load"]
	if svc == nil {
		t.Fatal("tree missing expectedload provider/service")
	}
	// All fixture declarations: 2 tf + 1 ts + 2 py + 2 go + 2 java/kt + 1 rs
	// + 1 errors = 11; the node_modules decoy is excluded.
	if len(svc.GetResources()) != 11 {
		t.Fatalf("got %d tree resources, want 11", len(svc.GetResources()))
	}

	var tfRes *treeResourceCheck
	for _, r := range svc.GetResources() {
		if r.GetDefinition().GetSource().GetFilename() == "terraform/main.tf" && r.GetDefinition().GetSource().GetStartLine() == 1 {
			tfRes = &treeResourceCheck{
				id:      r.GetId(),
				typ:     r.GetType(),
				resType: r.GetDefinition().GetResourceType(),
				monthly: r.GetAttributes().GetEntries()["monthly_requests"].GetIntValue(),
				conf:    r.GetAttributes().GetEntries()["confidence"].GetStringValue(),
			}
		}
	}
	if tfRes == nil {
		t.Fatal("terraform/main.tf:1 resource missing from tree")
	}
	if tfRes.typ != "expected_load" || tfRes.resType != "expected_load" {
		t.Errorf("types = %q/%q, want expected_load", tfRes.typ, tfRes.resType)
	}
	if tfRes.monthly != 5_000_000 {
		t.Errorf("monthly_requests attribute = %d, want 5000000", tfRes.monthly)
	}
	if tfRes.conf != "high" {
		t.Errorf("confidence attribute = %q, want high", tfRes.conf)
	}
	if tfRes.id == "" {
		t.Error("tree resource id must be set")
	}
}

type treeResourceCheck struct {
	id, typ, resType, conf string
	monthly                int64
}

func TestParseSingleFileTarget(t *testing.T) {
	// CloudFormation targets carry a single template path; scanning one file
	// works and paths stay repo-relative.
	root := fixtures(t)
	resp, err := (&service{}).Parse(context.Background(), &api.ParseRequest{
		RepoDirectory:    root,
		WorkingDirectory: root,
		Target: &api.ParseRequestTarget{
			Value: &api.ParseRequestTarget_Cloudformation{Cloudformation: &cloudformation.Target{
				TemplatePath: filepath.Join("rust", "lib.rs"),
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	res := resp.GetResult().GetCloudformation().GetResources()
	if len(res) != 1 {
		t.Fatalf("got %d resources, want 1", len(res))
	}
	for id, r := range res {
		if r.GetSourceRange().GetFilename() != "rust/lib.rs" {
			t.Errorf("%s: filename = %q, want rust/lib.rs", id, r.GetSourceRange().GetFilename())
		}
	}
}

// Ensure the scan metadata the RPCs expose stays importable without gRPC —
// guards the plain-function layering the specs require.
var _ = scan.CanonicalName
