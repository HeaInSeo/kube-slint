package kubeutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollUntil_SucceedsImmediately(t *testing.T) {
	calls := 0
	err := pollUntil(context.Background(), 10*time.Millisecond, func() (bool, error) {
		calls++
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestPollUntil_SucceedsOnSecondTick(t *testing.T) {
	calls := 0
	err := pollUntil(context.Background(), 10*time.Millisecond, func() (bool, error) {
		calls++
		return calls >= 2, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestPollUntil_RetryOnError(t *testing.T) {
	calls := 0
	err := pollUntil(context.Background(), 10*time.Millisecond, func() (bool, error) {
		calls++
		if calls < 3 {
			return false, errors.New("not ready")
		}
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestPollUntil_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	calls := 0
	err := pollUntil(ctx, 5*time.Millisecond, func() (bool, error) {
		calls++
		return false, nil // never succeeds
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled))
	assert.Greater(t, calls, 0)
}

func TestPollUntil_ContextCancelled_PreservesLastError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	sentinel := errors.New("transient: not ready")
	err := pollUntil(ctx, 5*time.Millisecond, func() (bool, error) {
		return false, sentinel
	})
	require.Error(t, err)
	// Both the transient error and ctx error must be present
	assert.True(t, errors.Is(err, sentinel), "last fn error must be preserved")
	assert.True(t,
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"ctx error must be present",
	)
}
