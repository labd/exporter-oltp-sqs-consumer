package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	factory := otlphttpexporter.NewFactory()

	// Initialize the traces exporter
	tracesConfig := &otlphttpexporter.Config{
		Encoding: "json",
		ClientConfig: confighttp.ClientConfig{
			Endpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
			Headers: getHeaders(
				"OTEL_EXPORTER_OTLP_HEADERS",
				"OTEL_EXPORTER_OTLP_TRACES_HEADERS",
			),
			// CustomRoundTripper: NewDebugTransport,
		},
	}
	traceExporter, err := factory.CreateTracesExporter(ctx, settings, tracesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	if err := traceExporter.Start(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to start trace exporter: %w", err)
	}

	// Initialize the metrics exporter
	metricsConfig := &otlphttpexporter.Config{
		Encoding: "json",
		ClientConfig: confighttp.ClientConfig{
			Endpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
			Headers: getHeaders(
				"OTEL_EXPORTER_OTLP_HEADERS",
				"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
			),
			// CustomRoundTripper: NewDebugTransport,
		},
	}
	metricsExporter, err := factory.CreateMetricsExporter(ctx, settings, metricsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	if err := metricsExporter.Start(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to start metric exporter: %w", err)
	}

	return &Exporters{
		traces:  traceExporter,
		metrics: metricsExporter,
	}, nil
}

func createHandler(exporters *Exporters) func(context.Context, events.SQSEvent) error {
	return func(ctx context.Context, sqsEvent events.SQSEvent) error {
		for _, message := range sqsEvent.Records {
			if err := processMessage(ctx, exporters, message); err != nil {
				log.Printf("failed to process message: %v\n", err)
			}
		}

		return nil
	}
}

func processMessage(ctx context.Context, exporters *Exporters, message events.SQSMessage) error {
	item, err := parseMessage(message)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	if item.Kind == "trace" {
		traces, err := decodeTraces(item.Data)
		if err != nil {
			return fmt.Errorf("failed to decode trace batch: %w", err)
		}

		if err := exporters.traces.ConsumeTraces(ctx, traces); err != nil {
			return fmt.Errorf("failed to upload trace: %v\n", err)
		}
	}

	if item.Kind == "metric" {
		metrics, err := decodeMetrics(item.Data)
		if err != nil {
			return fmt.Errorf("failed to decode metric item: %w", err)
		}

		if err := exporters.metrics.ConsumeMetrics(ctx, metrics); err != nil {
			return fmt.Errorf("failed to upload metric: %v\n", err)
		}
	}
	return nil
}

func parseMessage(message events.SQSMessage) (*ExportedItem, error) {
	var item ExportedItem
	if err := json.Unmarshal([]byte(message.Body), &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w\n", err)
	}

	b, err := decompressData(item.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w\n", err)
	}
	item.Data = string(b)

	return &item, nil
}

func getHeaders(keys ...string) map[string]configopaque.String {
	headers := make(map[string]configopaque.String)
	headers["User-Agent"] = configopaque.String("labd/exporter-oltp-sqs-consumer")

	for _, key := range keys {
		value := os.Getenv(key)
		if value != "" {
			parts := strings.Split(value, ",")
			for _, part := range parts {
				keyValue := strings.SplitN(part, "=", 2)
				if len(keyValue) == 2 {
					headers[keyValue[0]] = configopaque.String(keyValue[1])
				}
			}
		}
	}
	return headers
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
	ctx := context.Background()
	exporters, err := initExporters(ctx)
	if err != nil {
		panic(err)
	}

	handler := createHandler(exporters)
	lambda.Start(handler)
}
