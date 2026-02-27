package progress

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

func RunCLISpinner(w io.Writer, message string, fn func() error) error {
	spin := DotSpinner()
	if len(spin.Frames) == 0 {
		return fn()
	}

	ticker := time.NewTicker(spin.FPS)
	defer ticker.Stop()

	stop := make(chan struct{})
	done := make(chan struct{})
	var maxLen int

	go func() {
		defer close(done)

		var frame int
		for {
			line := fmt.Sprintf("%s %s", spin.Frames[frame], message)
			if len(line) > maxLen {
				maxLen = len(line)
			}
			_, _ = fmt.Fprintf(w, "\r%s", line)

			frame = (frame + 1) % len(spin.Frames)

			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	}()

	err := fn()
	close(stop)
	<-done

	if maxLen > 0 {
		_, _ = fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", maxLen))
	}
	return err
}

func DotSpinner() spinner.Spinner {
	return spinner.Spinner{
		Frames: append([]string(nil), spinner.Dot.Frames...),
		FPS:    spinner.Dot.FPS,
	}
}
