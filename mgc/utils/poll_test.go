package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPollImmediateSuccess(t *testing.T) {
	calls := 0
	err := Poll(context.Background(), time.Second, time.Hour, func(context.Context) (bool, error) { calls++; return true, nil })
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

func TestPollNeverReportsTimeoutAsSuccess(t *testing.T) {
	calls := 0
	err := Poll(context.Background(), 100*time.Millisecond, time.Hour, func(ctx context.Context) (bool, error) {
		_, ok := ctx.Deadline()
		require.True(t, ok)
		calls++
		return false, nil
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, calls)
}

func TestPollCancellationInterruptsWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := Poll(ctx, time.Hour, time.Hour, func(context.Context) (bool, error) { cancel(); return false, nil })
	require.ErrorIs(t, err, context.Canceled)
}

func TestPollDoesNotCallAPIAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Poll(ctx, time.Hour, time.Hour, func(context.Context) (bool, error) { t.Fatal("API called after cancellation"); return true, nil })
	require.ErrorIs(t, err, context.Canceled)
}

func TestPollPropagatesFailure(t *testing.T) {
	want := errors.New("operation failed")
	err := Poll(context.Background(), time.Second, time.Hour, func(context.Context) (bool, error) { return false, want })
	require.ErrorIs(t, err, want)
}
