package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	exporter := otlptracehttp.NewClient()
	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)

		trace, err := decodeTraceBatch(message)
		if err != nil {
			return fmt.Errorf("failed to decode trace batch: %w", err)
		}

		if spans := trace.GetResourceSpans(); spans != nil {
			err := exporter.UploadTraces(ctx, spans)
			if err != nil {
				fmt.Printf("failed to upload trace: %v\n", err)
			}
		}
	}

	return nil
}

func decodeTraceBatch(msg events.SQSMessage) (*tracepb.TracesData, error) {
	var traceBatch tracepb.TracesData
	err := protojson.Unmarshal([]byte(msg.Body), &traceBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace batch: %w", err)
	}
	return &traceBatch, nil
}

func main() {
	lambda.Start(handler)
}
