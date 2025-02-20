package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/build"
	"github.com/sashabaranov/go-openai"
)

var openAIModels = []Model{ModelBase, ModelLarge}

type openAIModelHealth struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type openAIHealthDetails struct {
	Configured bool                        `json:"configured"`
	OK         bool                        `json:"ok"`
	Error      string                      `json:"error,omitempty"`
	Models     map[Model]openAIModelHealth `json:"models"`
	Assistant  openAIModelHealth           `json:"assistant"`
}

type vectorHealthDetails struct {
	Enabled bool   `json:"enabled"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

type healthCheckDetails struct {
	OpenAI  openAIHealthDetails `json:"openAI"`
	Vector  vectorHealthDetails `json:"vector"`
	Version string              `json:"version"`
}

func getVersion() string {
	buildInfo, err := build.GetBuildInfo()
	if err != nil {
		return "unknown"
	}
	return buildInfo.Version
}

func (a *App) testOpenAIModel(ctx context.Context, model Model) error {
	llmProvider, err := createProvider(a.settings)
	if err != nil {
		return err
	}

	req := ChatCompletionRequest{
		Model: model,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: "Hello"},
			},
			MaxTokens: 1,
		},
	}
	_, err = llmProvider.ChatCompletion(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (a *App) testOpenAIAssistant(ctx context.Context) error {
	llmProvider, err := createProvider(a.settings)
	if err != nil {
		return err
	}

	limit := 1
	_, err = llmProvider.ListAssistants(ctx, &limit, nil, nil, nil)
	return err
}

// openAIHealth checks the health of the OpenAI configuration and caches the
// result if successful. The caller must lock a.healthCheckMutex.
func (a *App) openAIHealth(ctx context.Context) (openAIHealthDetails, error) {
	if a.healthOpenAI != nil {
		return *a.healthOpenAI, nil
	}

	// If OpenAI is disabled it has been configured but cannot be queried.
	if a.settings.OpenAI.Disabled {
		return openAIHealthDetails{
			OK:         false,
			Configured: true,
			Error:      "LLM functionality is disabled",
		}, nil
	}

	d := openAIHealthDetails{
		OK:         true,
		Configured: a.settings.OpenAI.Configured(),
		Models:     map[Model]openAIModelHealth{},
		Assistant:  openAIModelHealth{OK: false, Error: "Assistant not available"},
	}

	for _, model := range openAIModels {
		health := openAIModelHealth{OK: false, Error: "OpenAI not configured"}
		if d.Configured {
			health.OK = true
			health.Error = ""
			err := a.testOpenAIModel(ctx, model)
			if err != nil {
				health.OK = false
				health.Error = err.Error()
			}
		}
		d.Models[model] = health
	}
	anyOK := false
	for _, v := range d.Models {
		if v.OK {
			anyOK = true
			break
		}
	}
	if !anyOK {
		d.OK = false
		d.Error = "No functioning models are available"
	}

	if d.Configured {
		err := a.testOpenAIAssistant(ctx)
		if err == nil {
			d.Assistant.OK = true
			d.Assistant.Error = ""
		} else {
			d.Assistant.OK = false
			d.Assistant.Error = strings.Join([]string{d.Assistant.Error, err.Error()}, ": ")
		}
	}

	// Only cache result if openAI is ok to use.
	if d.OK {
		a.healthOpenAI = &d
	}
	return d, nil
}

// testVectorService checks the health of VectorAPI and caches the result if
// successful. The caller must lock a.healthCheckMutex.
func (a *App) testVectorService(ctx context.Context) error {
	if a.vectorService == nil {
		return fmt.Errorf("vector service not configured")
	}
	err := a.vectorService.Health(ctx)
	if err != nil {
		return fmt.Errorf("vector service health check failed: %w", err)
	}
	return nil
}

func (a *App) vectorHealth(ctx context.Context) vectorHealthDetails {
	if a.healthVector != nil {
		return *a.healthVector
	}

	d := vectorHealthDetails{
		Enabled: a.settings.Vector.Enabled,
		OK:      true,
	}
	if !d.Enabled {
		d.OK = false
		return d
	}
	err := a.testVectorService(ctx)
	if err != nil {
		d.OK = false
		d.Error = err.Error()
	}

	// Only cache if the health check succeeded.
	if d.OK {
		a.healthVector = &d
	}
	return d
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// It returns whether each feature is working based on the plugin settings.
func (a *App) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	a.healthCheckMutex.Lock()
	defer a.healthCheckMutex.Unlock()

	openAI, err := a.openAIHealth(ctx)
	if err != nil {
		openAI.OK = false
		openAI.Error = err.Error()
	}

	vector := a.vectorHealth(ctx)
	if vector.Error == "" {
		a.healthVector = &vector
	}

	details := healthCheckDetails{
		OpenAI:  openAI,
		Vector:  vector,
		Version: getVersion(),
	}
	body, err := json.Marshal(details)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "failed to marshal details",
		}, nil
	}
	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		JSONDetails: body,
	}, nil
}
