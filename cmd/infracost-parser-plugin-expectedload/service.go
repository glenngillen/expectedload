package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/infracost/expectedload/internal/scan"
	pluginpb "github.com/infracost/proto/gen/go/infracost/plugin"
)

// service implements the shared PluginService and ParserService. It holds no state: the CLI may
// run several plugin processes concurrently, and every RPC works purely on its
// request.
type service struct {
	pluginpb.UnimplementedPluginServiceServer
	pluginpb.UnimplementedParserServiceServer
}

func (s *service) GetPluginInfo(_ context.Context, _ *pluginpb.GetPluginInfoRequest) (*pluginpb.GetPluginInfoResponse, error) {
	return &pluginpb.GetPluginInfoResponse{
		Type: pluginpb.PluginType_PARSER, Name: scan.CanonicalName, Version: version,
		Description: "Extract expected-load declarations from source-code comments",
		Url:         "https://github.com/infracost/expectedload", Author: "Infracost",
	}, nil
}

func (s *service) GetParserConfig(_ context.Context, _ *pluginpb.GetParserConfigRequest) (*pluginpb.GetParserConfigResponse, error) {
	projectType := scan.ProjectType
	return &pluginpb.GetParserConfigResponse{IdentificationPriority: scan.Priority, ConfigFileProjectType: &projectType}, nil
}

func (s *service) IdentifyProjects(_ context.Context, req *pluginpb.IdentifyProjectsRequest) (*pluginpb.IdentifyProjectsResponse, error) {
	if req == nil || req.GetDirectory() == "" {
		return &pluginpb.IdentifyProjectsResponse{}, nil
	}
	entries, err := os.ReadDir(req.GetDirectory())
	if err != nil {
		return &pluginpb.IdentifyProjectsResponse{}, nil
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && scan.DetectPath(filepath.Join(req.GetDirectory(), entry.Name())) {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return &pluginpb.IdentifyProjectsResponse{Files: files}, nil
}

func (s *service) Parse(_ context.Context, req *pluginpb.ParseRequest) (*pluginpb.ParseResponse, error) {
	if req == nil || req.GetPath() == "" {
		return nil, fmt.Errorf("path is required")
	}
	res := scan.Scan(req.GetPath())
	return &pluginpb.ParseResponse{Diagnostics: toProtoDiagnostics(res.Diagnostics, ""), Tree: toTree(res, "")}, nil
}
