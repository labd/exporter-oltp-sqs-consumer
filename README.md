# OpenTelemetry SQS consumer
This is a Lambda function to consume traces generated by for example https://github.com/labd/otel-exporter-oltp-sqs-js 
to push traces to an OpenTelemetry collector/endpoint.

It supports the regular OpenTelemetry environment variables to specify the collector endpoint 
