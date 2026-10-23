// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package transformfuncs

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

func TestDropOriginalPayloadThroughTransformProcessor(t *testing.T) {
	factory := NewTransformFactory().(xprocessor.Factory)
	cfg := factory.CreateDefaultConfig().(*transformprocessor.Config)
	require.NoError(t, confmap.NewFromStringMap(map[string]any{
		"error_mode": "ignore",
		"profile_statements": []any{map[string]any{
			"context":    "profile",
			"statements": []any{`drop_original_payload() where profile.original_payload_format != ""`},
		}},
	}).Unmarshal(cfg))
	require.NoError(t, cfg.Validate())

	sink := new(consumertest.ProfilesSink)
	proc, err := factory.CreateProfiles(context.Background(), processortest.NewNopSettings(factory.Type()), cfg, sink)
	require.NoError(t, err)

	pd := pprofile.NewProfiles()
	profiles := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
	withPayload := profiles.AppendEmpty()
	withPayload.SetOriginalPayloadFormat("jfr")
	withPayload.OriginalPayload().FromRaw([]byte("original-jfr-bytes"))
	withPayload.SetDurationNano(7)
	plain := profiles.AppendEmpty()
	plain.SetDurationNano(9)

	require.NoError(t, proc.ConsumeProfiles(context.Background(), pd))
	require.Len(t, sink.AllProfiles(), 1)
	out := sink.AllProfiles()[0].ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles()
	require.Equal(t, 2, out.Len())
	assert.Equal(t, 0, out.At(0).OriginalPayload().Len())
	assert.Equal(t, "", out.At(0).OriginalPayloadFormat())
	assert.Equal(t, uint64(7), out.At(0).DurationNano(), "other fields untouched")
	assert.Equal(t, uint64(9), out.At(1).DurationNano())
}
