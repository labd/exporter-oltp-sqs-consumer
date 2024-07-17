package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitExporters(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	ctx := context.Background()
	exporters, err := initExporters(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, exporters)
}

func TestDeserialize(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	ctx := context.Background()
	exporters, err := initExporters(ctx)

	data := `
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": {
              "stringValue": "my-test-service"
            }
          },
          {
            "key": "process.command",
            "value": {
              "stringValue": "/app/run.ts"
            }
          }
        ],
        "droppedAttributesCount": 0
      },
      "scopeSpans": [
        {
          "scope": {
            "name": "@opentelemetry/instrumentation-dns",
            "version": "0.37.0"
          },
          "spans": [
            {
              "traceId": "d608bd09d0c84fb2e79f07c222e9682b",
              "spanId": "8d652619dfa663ca",
              "name": "dns.lookup",
              "kind": 3,
              "startTimeUnixNano": "1721147595962000000",
              "endTimeUnixNano": "1721147595967767209",
              "attributes": [
                {
                  "key": "peer.ipv6",
                  "value": {
                    "stringValue": "::1"
                  }
                }
              ],
              "droppedAttributesCount": 0,
              "events": [],
              "droppedEventsCount": 0,
              "status": {
                "code": 0
              },
              "links": [],
              "droppedLinksCount": 0
            }
          ]
        }
      ]
    }
  ]
}
	`

	trace, err := decodeTraces(data)
	assert.NoError(t, err)
	assert.NotNil(t, trace)

	exporters.traces.ConsumeTraces(ctx, trace)
}

func TestMetricData(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	ctx := context.Background()
	exporters, err := initExporters(ctx)

	data := `
  {
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          {
            "key": "telemetry.sdk.language",
            "value": {
              "stringValue": "nodejs"
            }
          },
          {
            "key": "telemetry.sdk.name",
            "value": {
              "stringValue": "opentelemetry"
            }
          },
          {
            "key": "telemetry.sdk.version",
            "value": {
              "stringValue": "1.25.0"
            }
          },
          {
            "key": "process.command",
            "value": {
              "stringValue": "/app/run.ts"
            }
          }
        ],
        "droppedAttributesCount": 0
      },
      "scopeMetrics": []
    }
  ]
}`

	metrics, err := decodeMetrics(data)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)

	exporters.metrics.ConsumeMetrics(ctx, metrics)
}
