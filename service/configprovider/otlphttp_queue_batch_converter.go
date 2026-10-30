// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package configprovider

import (
	"context"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

// otlphttpQueueBatchConverter deletes a null `sending_queue.batch` key from
// otlphttp exporters at load time. An explicit sending_queue makes the
// serializer emit `batch: null`, which the collector would otherwise promote
// to the default batch config, re-enabling the exporter batcher.
type otlphttpQueueBatchConverter struct{}

// NewOTLPHTTPQueueBatchConverterFactory returns a factory for the converter.
func NewOTLPHTTPQueueBatchConverterFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(func(_ confmap.ConverterSettings) confmap.Converter {
		return &otlphttpQueueBatchConverter{}
	})
}

// Convert drops `sending_queue.batch` when it is null on any otlphttp exporter.
func (c *otlphttpQueueBatchConverter) Convert(_ context.Context, conf *confmap.Conf) error {
	raw := conf.ToStringMap()
	exporters, ok := raw["exporters"].(map[string]any)
	if !ok {
		return nil
	}
	changed := false
	for name, cfg := range exporters {
		exporterType, _, _ := strings.Cut(name, "/")
		if exporterType != "otlphttp" && exporterType != "otlp_http" {
			continue
		}
		exporterCfg, ok := cfg.(map[string]any)
		if !ok {
			continue
		}
		sendingQueue, ok := exporterCfg["sending_queue"].(map[string]any)
		if !ok {
			continue
		}
		if batch, present := sendingQueue["batch"]; present && batch == nil {
			delete(sendingQueue, "batch")
			changed = true
		}
	}
	if !changed {
		return nil
	}
	// confmap has no public delete that reaches a null leaf, so rebuild from the map.
	*conf = *confmap.NewFromStringMap(raw)
	return nil
}
