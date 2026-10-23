// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

// Package transformfuncs registers agent-specific OTTL functions on the upstream
// transform processor. drop_original_payload() clears Profile.original_payload and
// Profile.original_payload_format together: OTTL has no empty-bytes literal, so a
// stock set() statement cannot clear the bytes, and the backend rejects a profile
// whose format and payload are not both set or both unset.
package transformfuncs

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlprofile"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"go.opentelemetry.io/collector/processor"
)

// NewTransformFactory returns the upstream transform processor factory with the
// agent's profile-context functions added to the default set.
func NewTransformFactory() processor.Factory {
	return transformprocessor.NewFactoryWithOptions(
		transformprocessor.WithProfileFunctionsNew(ProfileFunctions()),
	)
}

// ProfileFunctions is the default OTTL profile function set plus drop_original_payload.
func ProfileFunctions() []ottl.Factory[*ottlprofile.TransformContext] {
	funcs := []ottl.Factory[*ottlprofile.TransformContext]{newDropOriginalPayloadFactory()}
	for _, f := range ottlfuncs.StandardFuncs[*ottlprofile.TransformContext]() {
		funcs = append(funcs, f)
	}
	return funcs
}

func newDropOriginalPayloadFactory() ottl.Factory[*ottlprofile.TransformContext] {
	return ottl.NewFactory("drop_original_payload", nil, createDropOriginalPayload)
}

func createDropOriginalPayload(_ ottl.FunctionContext, _ ottl.Arguments) (ottl.ExprFunc[*ottlprofile.TransformContext], error) {
	return func(_ context.Context, tCtx *ottlprofile.TransformContext) (any, error) {
		p := tCtx.GetProfile()
		p.OriginalPayload().FromRaw(nil)
		p.SetOriginalPayloadFormat("")
		return nil, nil
	}, nil
}
