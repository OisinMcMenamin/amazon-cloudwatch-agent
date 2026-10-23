// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package otlp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/aws/amazon-cloudwatch-agent/internal/transformfuncs"
	otlpreceiver "github.com/aws/amazon-cloudwatch-agent/translator/translate/otel/receiver/otlp"
)

// TestProfilesDropOriginalPayloadBehavior builds transform/drop_original_payload from the translated
// config with the agent's transform factory and checks that a converter's source bytes leave the
// pipeline cleared while the rest of the profile is untouched.
func TestProfilesDropOriginalPayloadBehavior(t *testing.T) {
	t.Cleanup(otlpreceiver.ClearConfigCache)
	resetGlobalConfig(t, "us-east-1")
	setMode(t, "ec2")
	setKubernetesMode(t, "")
	setECS(t, false)
	conf := confmap.NewFromStringMap(otlpSectionWithClusterNameConf)
	got, err := (&profilesPipelineTranslator{}).Translate(conf)
	require.NoError(t, err)

	loaded := loadProcessorsAsAgent(t, got, conf)
	sink := new(consumertest.ProfilesSink)
	proc := newProfilesProcessor(t, transformfuncs.NewTransformFactory(), loaded, "transform", dropOriginalPayloadName, sink)

	profiles := pprofile.NewProfiles()
	scopeProfiles := profiles.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty()
	converted := scopeProfiles.Profiles().AppendEmpty()
	converted.SetOriginalPayloadFormat("jfr")
	converted.OriginalPayload().FromRaw([]byte("original-jfr-bytes"))
	converted.SetDurationNano(7)
	converted.Samples().AppendEmpty().Values().Append(3)
	native := scopeProfiles.Profiles().AppendEmpty()
	native.SetDurationNano(9)

	require.NoError(t, proc.ConsumeProfiles(context.Background(), profiles))
	require.Len(t, sink.AllProfiles(), 1)
	out := sink.AllProfiles()[0].ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles()
	require.Equal(t, 2, out.Len())
	assert.Equal(t, 0, out.At(0).OriginalPayload().Len())
	assert.Equal(t, "", out.At(0).OriginalPayloadFormat())
	assert.Equal(t, uint64(7), out.At(0).DurationNano())
	assert.Equal(t, int64(3), out.At(0).Samples().At(0).Values().At(0))
	assert.Equal(t, uint64(9), out.At(1).DurationNano())
}
