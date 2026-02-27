package progress

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotSpinner(t *testing.T) {
	t.Parallel()

	spinA := DotSpinner()
	spinB := DotSpinner()

	assert.NotEmpty(t, spinA.Frames)
	assert.Greater(t, spinA.FPS, time.Duration(0))

	spinA.Frames[0] = "x"
	assert.NotEqual(t, spinA.Frames[0], spinB.Frames[0])
}

func TestRunCLISpinner(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	err := RunCLISpinner(buf, "processing", func() error {
		time.Sleep(2 * DotSpinner().FPS)
		return nil
	})

	require.NoError(t, err)
	assert.True(t, strings.Contains(buf.String(), "processing"))
}

func TestRunCLISpinnerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	err := RunCLISpinner(&bytes.Buffer{}, "processing", func() error {
		return wantErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}
