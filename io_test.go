// Copyright 2026 Serge Smertin
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nfx/go-tui/internal/assert"
)

func chainIOforTest(t *testing.T, width, height int) (*chanIO, *writeC) {
	ctx, cancel := context.WithCancel(t.Context())
	realOut := newWriteC(ctx)
	cio := newUnstartedIO(ctx, width, height)
	go cio.handleViewports(ctx)
	go cio.forwardTo(ctx, realOut)
	t.Cleanup(func() {
		cancel()
		close(cio.In)
		close(cio.Out)
		close(realOut.C)
	})

	return cio, realOut
}

func TestChanIO_Forward(t *testing.T) {
	t.SkipNow()
	cio, stdout := chainIOforTest(t, 12, 3)

	// write 5 lines
	_, err := fmt.Fprint(cio, "a\nb\nc\nd\ne\n")
	assert.NoError(t, err)

	// render 3 lines due to the viewport height
	assert.Equal(t, "\rc           \n\rd           \n\re           ", <-stdout.C)

	// write one more line
	_, err = fmt.Fprint(cio, "f\n")
	assert.NoError(t, err)

	// and have the previous two lines still rendered
	assert.Equal(t,
		"\x1b[3A\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rd           \n\re           \n\rf           ",
		<-stdout.C)

	ticks := make(chan time.Time)
	defer close(ticks)
	tick := func() {
		go func() {
			select {
			case <-cio.ctx.Done():
			case ticks <- time.Now():
			}
		}()
	}

	s, err := NewSpinners(
		WithOutput(cio),
		WithContext(cio.ctx),
		spinnersOpt(func(s *Spinners) error {
			s.ticks = ticks

			return nil
		}),
	)
	assert.NoError(t, err)

	s.MustAddBackground().Update("s: A")
	tick()

	assert.Equal(t,
		"\x1b[3A\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\r\r... s: A   \n\r\r           \n\rf           ",
		<-stdout.C)

	// write one more line
	_, err = fmt.Fprint(cio, "g\n")
	assert.NoError(t, err)

	assert.Equal(t,
		"\x1b[3A\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\r\r... s: A   \n\r\r           \n\rg           ",
		<-stdout.C)
}
