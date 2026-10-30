// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package configprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
)

func sendingQueueWithNullBatch(queueSize int) map[string]any {
	return map[string]any{
		"batch":             nil,
		"block_on_overflow": false,
		"num_consumers":     10,
		"queue_size":        queueSize,
		"sizer":             nil,
		"storage":           nil,
		"wait_for_result":   false,
	}
}

func TestOTLPHTTPQueueBatchConverter(t *testing.T) {
	ctx := context.Background()
	conv := &otlphttpQueueBatchConverter{}

	t.Run("strips null batch on otlphttp exporter", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"exporters": map[string]any{
				"otlp_http/profiles": map[string]any{
					"profiles_endpoint": "https://monitoring.us-east-1.amazonaws.com/v1development/profiles",
					"sending_queue":     sendingQueueWithNullBatch(64),
				},
			},
		})
		require.NoError(t, conv.Convert(ctx, conf))

		sq, ok := conf.Get("exporters").(map[string]any)["otlp_http/profiles"].(map[string]any)["sending_queue"].(map[string]any)
		require.True(t, ok)
		_, present := sq["batch"]
		assert.False(t, present, "null batch key should be removed")
		assert.Equal(t, 64, sq["queue_size"], "queue_size must be preserved")
	})

	t.Run("reloaded exporter keeps the batcher off and the cap", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"exporters": map[string]any{
				"otlp_http/profiles": map[string]any{
					"profiles_endpoint": "https://monitoring.us-east-1.amazonaws.com/v1development/profiles",
					"sending_queue":     sendingQueueWithNullBatch(64),
				},
			},
		})
		require.NoError(t, conv.Convert(ctx, conf))

		sub, err := conf.Sub("exporters::otlp_http/profiles")
		require.NoError(t, err)
		cfg := otlphttpexporter.NewFactory().CreateDefaultConfig().(*otlphttpexporter.Config)
		require.NoError(t, sub.Unmarshal(cfg))

		require.True(t, cfg.QueueConfig.HasValue())
		q := cfg.QueueConfig.Get()
		assert.EqualValues(t, 64, q.QueueSize, "cap preserved")
		assert.Equal(t, "requests", q.Sizer.String(), "request sizer preserved")
		assert.False(t, q.Batch.HasValue(), "batcher must stay off")
	})

	t.Run("no-op when batch is absent", func(t *testing.T) {
		sq := sendingQueueWithNullBatch(64)
		delete(sq, "batch")
		conf := confmap.NewFromStringMap(map[string]any{
			"exporters": map[string]any{
				"otlp_http/profiles": map[string]any{"sending_queue": sq},
			},
		})
		require.NoError(t, conv.Convert(ctx, conf))
		got := conf.Get("exporters").(map[string]any)["otlp_http/profiles"].(map[string]any)["sending_queue"].(map[string]any)
		_, present := got["batch"]
		assert.False(t, present)
	})

	t.Run("leaves a non-null batch untouched", func(t *testing.T) {
		sq := sendingQueueWithNullBatch(64)
		sq["batch"] = map[string]any{"flush_timeout": "200ms"}
		conf := confmap.NewFromStringMap(map[string]any{
			"exporters": map[string]any{
				"otlp_http/profiles": map[string]any{"sending_queue": sq},
			},
		})
		require.NoError(t, conv.Convert(ctx, conf))
		got := conf.Get("exporters").(map[string]any)["otlp_http/profiles"].(map[string]any)["sending_queue"].(map[string]any)
		assert.NotNil(t, got["batch"], "explicit batch config must be preserved")
	})

	t.Run("leaves sibling otlphttp exporters untouched", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"exporters": map[string]any{
				"otlp_http/profiles": map[string]any{
					"profiles_endpoint": "https://monitoring.us-east-1.amazonaws.com/v1development/profiles",
					"sending_queue":     sendingQueueWithNullBatch(64),
				},
				"otlp_http/metrics": map[string]any{
					"metrics_endpoint": "https://monitoring.us-east-1.amazonaws.com/v1/metrics",
					"compression":      "gzip",
				},
			},
		})
		require.NoError(t, conv.Convert(ctx, conf))

		exporters := conf.Get("exporters").(map[string]any)
		profilesSQ := exporters["otlp_http/profiles"].(map[string]any)["sending_queue"].(map[string]any)
		_, present := profilesSQ["batch"]
		assert.False(t, present, "profiles null batch stripped")
		assert.Equal(t, 64, profilesSQ["queue_size"])

		metrics := exporters["otlp_http/metrics"].(map[string]any)
		assert.Equal(t, "https://monitoring.us-east-1.amazonaws.com/v1/metrics", metrics["metrics_endpoint"])
		assert.Equal(t, "gzip", metrics["compression"])
		_, hasSQ := metrics["sending_queue"]
		assert.False(t, hasSQ, "sibling exporter without a sending_queue is left untouched")
	})

	t.Run("ignores non-otlphttp exporters and missing exporters", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"exporters": map[string]any{
				"awscloudwatchlogs": map[string]any{"sending_queue": sendingQueueWithNullBatch(64)},
			},
		})
		require.NoError(t, conv.Convert(ctx, conf))
		got := conf.Get("exporters").(map[string]any)["awscloudwatchlogs"].(map[string]any)["sending_queue"].(map[string]any)
		_, present := got["batch"]
		assert.True(t, present, "non-otlphttp exporters are left alone")

		empty := confmap.NewFromStringMap(map[string]any{})
		require.NoError(t, conv.Convert(ctx, empty))
	})
}
