package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/infracost/expectedload/internal/scan"
	"github.com/infracost/proto/gen/go/infracost/parser"
	"github.com/infracost/proto/gen/go/infracost/parser/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// service implements api.ParserServiceServer. It holds no state: the CLI may
// run several plugin processes concurrently, and every RPC works purely on its
// request.
type service struct {
	api.UnimplementedParserServiceServer
}

func (s *service) Describe(_ context.Context, _ *api.DescribeRequest) (*api.DescribeResponse, error) {
	return &api.DescribeResponse{
		Name:                scan.CanonicalName,
		DisplayName:         scan.DisplayName,
		Priority:            scan.Priority,
		FileExtensions:      scan.FileExtensions(),
		SupportsDirectories: scan.SupportsDirectories,
	}, nil
}

func (s *service) Detect(_ context.Context, req *api.DetectRequest) (*api.DetectResponse, error) {
	notDetected := &api.DetectResponse{Detected: false}
	if req == nil || req.GetPath() == "" {
		return notDetected, nil
	}

	var detected bool
	if req.GetContentProvided() {
		// LSP virtual document: sniff the provided bytes, never touch disk.
		// The extension gate still applies — content is only set for files.
		if _, ok := scan.SyntaxForPath(req.GetPath()); ok {
			detected = scan.DetectContent(req.GetContent())
		}
	} else {
		detected = scan.DetectPath(req.GetPath())
	}
	if !detected {
		return notDetected, nil
	}

	return &api.DetectResponse{
		Detected:    true,
		ProjectType: scan.ProjectType,
		// MEDIUM: the marker is a strong content sniff, but the extensions are
		// generic source files owned by other tools first.
		Confidence: api.DetectConfidence_DETECT_CONFIDENCE_MEDIUM,
	}, nil
}

func (s *service) Initialize(_ context.Context, _ *api.InitializeRequest) (*api.InitializeResponse, error) {
	// Expected-load declarations are usage annotations, not costable
	// resources, so the supported-resources lists don't change our output.
	return &api.InitializeResponse{}, nil
}

func (s *service) Parse(_ context.Context, req *api.ParseRequest) (*api.ParseResponse, error) {
	if req == nil || req.GetTarget() == nil {
		return nil, status.Error(codes.InvalidArgument, "parse request and target must not be nil")
	}

	res, prefix := scanTarget(req.GetRepoDirectory(), req.GetWorkingDirectory(), req.GetTarget())
	resp := &api.ParseResponse{Diagnostics: toProtoDiagnostics(res.Diagnostics, prefix)}

	// The pinned proto's ParseResponseResult oneof has no expected-load
	// variant; populate the variant matching the requested target so the CLI
	// receives a well-formed response, per specs/parse-output-mapping.md.
	// ParseToTree is the primary, variant-free output path.
	switch req.GetTarget().GetValue().(type) {
	case *api.ParseRequestTarget_Terraform:
		resp.Result = &api.ParseResponseResult{
			Value: &api.ParseResponseResult_Terraform{Terraform: toTerraformResult(res, prefix)},
		}
	case *api.ParseRequestTarget_Cloudformation:
		resp.Result = &api.ParseResponseResult{
			Value: &api.ParseResponseResult_Cloudformation{Cloudformation: toCloudformationResult(res, prefix)},
		}
	default:
		// Recoverable: a target variant with no matching result variant
		// (e.g. terragrunt) still gets its diagnostics and an empty result.
		resp.Diagnostics = append(resp.Diagnostics, &parser.Diagnostic{
			Error:   "expected-load has no result mapping for this target type; use ParseToTree",
			Warning: true,
		})
	}
	return resp, nil
}

func (s *service) ParseToTree(_ context.Context, req *api.ParseToTreeRequest) (*api.ParseToTreeResponse, error) {
	if req == nil || req.GetTarget() == nil {
		return nil, status.Error(codes.InvalidArgument, "parse request and target must not be nil")
	}

	res, prefix := scanTarget(req.GetRepoDirectory(), req.GetWorkingDirectory(), req.GetTarget())
	return &api.ParseToTreeResponse{
		Diagnostics: toProtoDiagnostics(res.Diagnostics, prefix),
		Tree:        toTree(res, prefix),
	}, nil
}

// scanTarget resolves the path to scan from the request and runs the scan.
// It returns the scan result plus the prefix that rebases result-relative
// paths onto repo_directory (declaration locations are reported relative to
// the repo root).
func scanTarget(repoDir, workingDir string, target *api.ParseRequestTarget) (scan.Result, string) {
	root := workingDir
	switch t := target.GetValue().(type) {
	case *api.ParseRequestTarget_Terraform:
		if d := t.Terraform.GetDirectory(); d != "" {
			root = resolve(repoDir, workingDir, d)
		}
	case *api.ParseRequestTarget_Cloudformation:
		if p := t.Cloudformation.GetTemplatePath(); p != "" {
			root = resolve(repoDir, workingDir, p)
		}
	}
	if root == "" {
		root = repoDir
	}

	// Declaration paths come back relative to the scan root (for a single
	// file: just its base name), but must be reported relative to
	// repo_directory — compute the prefix that rebases them.
	prefix := ""
	if repoDir != "" {
		base := root
		if info, err := os.Stat(root); err == nil && !info.IsDir() {
			base = filepath.Dir(root)
		}
		if rel, err := filepath.Rel(repoDir, base); err == nil && rel != "." && !isUpward(rel) {
			prefix = filepath.ToSlash(rel)
		}
	}
	return scan.Scan(root), prefix
}

// resolve interprets a target path that may be absolute, repo-relative, or
// working-directory-relative.
func resolve(repoDir, workingDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	base := workingDir
	if base == "" {
		base = repoDir
	}
	return filepath.Join(base, p)
}

func isUpward(rel string) bool {
	return rel == ".." || len(rel) > 2 && rel[:3] == ".."+string(filepath.Separator)
}
