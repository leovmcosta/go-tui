// Copyright 2026 Serge Smertin
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/nfx/go-tui/internal/assert"
)

func testIOforSpinners(t *testing.T, width, height int, o ...opt) (*chanIO, func(), opt) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cio := &chanIO{
		ctx: ctx,
		In:  make(chan string),
		Out: make(chan string),
	}
	ticks := make(chan time.Time)
	t.Cleanup(func() {
		cancel()
		// <-cio.Out // clear
		close(cio.In)
		close(cio.Out)
		close(ticks)
	})

	return cio, func() {
			go func() {
				select {
				case <-ctx.Done():
				case ticks <- time.Now():
				}
			}()
		}, WithOptions(append(opts{
			WithInput(cio),
			WithOutput(cio),
			WithContext(ctx),
			spinnersOpt(func(s *Spinners) error {
				s.ticks = ticks
				s.makeTermIO = func(in io.Reader, out io.Writer) (*termIO, error) {
					return &termIO{
						in:      in,
						out:     out,
						Width:   width,
						Height:  height,
						Restore: func() error { return nil },
					}, nil
				}

				return nil
			}),
		}, o...,
		)...)
}

func spinnersForTest(t *testing.T) (*Spinners, *chanIO, func()) {
	t.Helper()
	cio, tick, opts := testIOforSpinners(t, 12, 4)
	s, err := NewSpinners(opts)
	assert.NoError(t, err)

	return s, cio, tick
}

func TestNewSpinners(t *testing.T) {
	s, cio, tick := spinnersForTest(t)
	assert.NotNil(t, s)
	assert.NotNil(t, cio)

	// test that the spinner is created
	ctx, cancel := context.WithCancel(t.Context())
	first, err := s.Add(ctx)
	assert.NoError(t, err)

	first.Update("first: A")

	tick()
	assert.Equal(t, "\r... first: A\n\r", <-cio.Out)

	second := s.MustAddBackground()
	second.Update("second: A")

	tick()
	assert.Equal(t, "\x1b[1A\r\x1b[K\r .. first: A\n\r\r .. second: A\n\r", <-cio.Out)

	cancel()

	tick()
	assert.Equal(t, "\x1b[2A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1A\r\r  . first: A\n\r\r  . second: A\n\r", <-cio.Out)

	s.Close()
	// assert.Equal(t, "\x1b[2A", <-cio.Out)
}
