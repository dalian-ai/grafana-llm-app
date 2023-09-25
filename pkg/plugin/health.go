package plugin

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/build"
)

type healthCheckResponse struct {
	OpenAIEnabled bool   `json:"openAI"`
	VectorEnabled bool   `json:"vector"`
	Version       string `json:"version"`
}

func getVersion() string {
	buildInfo, err := build.GetBuildInfo()
	if err != nil {
		return "unknown"
	}
	return buildInfo.Version
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// It returns whether each feature is enabled based on the plugin settings.
func (a *App) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	settings := req.PluginContext.AppInstanceSettings
	resp := healthCheckResponse{
		OpenAIEnabled: settings.DecryptedSecureJSONData[openAIKey] != "",
		VectorEnabled: a.vectorService != nil,
		Version:       getVersion(),
	}
	body, err := json.Marshal(resp)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "failed to marshal response",
		}, nil
	}
	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		JSONDetails: body,
	}, nil
}