package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestDeserialize(t *testing.T) {

	message := events.SQSMessage{
		Body: `
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": {
              "stringValue": "my-service"
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
	`,
	}

	trace, err := decodeTraceBatch(message)
	assert.NoError(t, err)
	assert.NotNil(t, trace)
}
