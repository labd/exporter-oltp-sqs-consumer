package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type ExportedItem struct {
	Kind string `json:"kind"`
	Data string `json:"data"`
}

type Exporters struct {
	traces  exporter.Traces
	metrics exporter.Metrics
}

func initExporters(ctx context.Context) (*Exporters, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	settings := exporter.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger:         logger,
			MeterProvider:  noopmetric.NewMeterProvider(),
			TracerProvider: nooptrace.NewTracerProvider(),
		},
	}
	config := &otlphttpexporter.Config{
		Encoding: "json",
		ClientConfig: confighttp.ClientConfig{
			Endpoint:           os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
			Headers:            map[string]configopaque.String{},
			CustomRoundTripper: NewDebugTransport,
		},
	}

	// Parse the environment variable and add to headers map
	envHeaders := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")
	if envHeaders != "" {
		parts := strings.Split(envHeaders, ",")
		for _, part := range parts {
			keyValue := strings.SplitN(part, "=", 2)
			if len(keyValue) == 2 {
				config.ClientConfig.Headers[keyValue[0]] = configopaque.String(keyValue[1])
			}
		}
	}

	factory := otlphttpexporter.NewFactory()
	traceExporter, err := factory.CreateTracesExporter(ctx, settings, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	metricsExporter, err := factory.CreateMetricsExporter(ctx, settings, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	if err := traceExporter.Start(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to start trace exporter: %w", err)
	}

	if err := metricsExporter.Start(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to start metric exporter: %w", err)
	}

	return &Exporters{
		traces:  traceExporter,
		metrics: metricsExporter,
	}, nil
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {

	exporters, err := initExporters(ctx)
	if err != nil {
		panic(err)
	}

	for _, message := range sqsEvent.Records {
		var item ExportedItem
		if err := json.Unmarshal([]byte(message.Body), &item); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w\n", err)
		}

		if item.Kind == "trace" {
			traces, err := decodeTraces(item.Data)
			if err != nil {
				return fmt.Errorf("failed to decode trace batch: %w", err)
			}

			if err := exporters.traces.ConsumeTraces(ctx, traces); err != nil {
				fmt.Printf("failed to upload trace: %v\n", err)
			}
		}

		if item.Kind == "metric" {
			metrics, err := decodeMetrics(item.Data)
			if err != nil {
				return fmt.Errorf("failed to decode metric item: %w", err)
			}

			if err := exporters.metrics.ConsumeMetrics(ctx, metrics); err != nil {
				fmt.Printf("failed to upload metric: %v\n", err)
			}
		}

	}

	return nil
}

func decodeTraces(data string) (ptrace.Traces, error) {
	decoder := ptrace.JSONUnmarshaler{}
	return decoder.UnmarshalTraces([]byte(data))
}

func decodeMetrics(data string) (pmetric.Metrics, error) {
	decoder := pmetric.JSONUnmarshaler{}
	return decoder.UnmarshalMetrics([]byte(data))
}

func main() {
	lambda.Start(handler)
}
