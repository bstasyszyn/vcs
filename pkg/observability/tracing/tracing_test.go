/*
   Copyright SecureKey Technologies Inc.

   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package tracing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	t.Run("Provider NONE", func(t *testing.T) {
		tp, tracer, err := Initialize(ProviderNone, "service1", "")
		require.NoError(t, err)
		require.Nil(t, tp)
		require.NotNil(t, tracer)
	})

	t.Run("Provider JAEGER", func(t *testing.T) {
		tp, tracer, err := Initialize(ProviderJaeger, "service1", "")
		require.NoError(t, err)
		require.NotNil(t, tp)
		require.NotNil(t, tracer)
	})

	t.Run("Unsupported provider", func(t *testing.T) {
		tp, tracer, err := Initialize("unsupported", "service1", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported tracing provider")
		require.Nil(t, tp)
		require.Nil(t, tracer)
	})
}
