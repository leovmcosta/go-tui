// Copyright 2026 Serge Smertin
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"io"
	"testing"

	"github.com/nfx/go-tui/internal/assert"
)

func testIOforDropdown(t *testing.T, width, height int, o ...opt) (*chanIO, opt) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cio := &chanIO{
		ctx: ctx,
		In:  make(chan string),
		Out: make(chan string),
	}
	t.Cleanup(func() {
		cancel()
		close(cio.In)
		close(cio.Out)
	})

	return cio, WithOptions(append(opts{
		WithInput(cio),
		WithOutput(cio),
		WithContext(ctx),
		dropdownOpt(func(d *dropdown) error {
			d.makeTermIO = func(in io.Reader, out io.Writer) (*termIO, error) {
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
		WithLabelTemplate("{{ . }}"),
		WithActiveItemTemplate("+ {{ . }}"),
		WithInactiveItemTemplate("- {{ . }}"),
		WithMoreItemsTemplate("~ {{ .More }} of {{ .Total }} more"),
		WithAnswerTemplate("{{ .Label }}: {{ .Answer }}"),
	}, o...,
	)...)
}

func confirmForTest(t *testing.T) (in, out chan string, result chan bool) {
	t.Helper()
	cio, opts := testIOforDropdown(t, 80, 120)
	result = make(chan bool)
	go func() {
		defer close(result)
		result <- Confirm("Are you sure?", opts)
	}()

	return cio.In, cio.Out, result
}

func TestSimpleCase(t *testing.T) {
	in, out, res := confirmForTest(t)
	assert.Equal(t,
		"\rAre you sure? + Yes\n\r              - No\n\r",
		<-out)
	in <- "\x0d" // enter
	assert.Equal(t, "\x1b[2A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1A\r", <-out)
	assert.Equal(t, "Are you sure?: Yes\n", <-out)
	assert.Equal(t, true, <-res)
}

func TestDenyCase(t *testing.T) {
	in, out, res := confirmForTest(t)
	assert.Equal(t,
		"\rAre you sure? + Yes\n\r              - No\n\r",
		<-out)
	in <- "\x1b\x5b\x42" // down
	assert.Equal(t,
		"\x1b[2A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1A\r\rAre you sure? - Yes\n\r              + No\n\r",
		<-out)
	in <- "\x0d" // enter
	assert.Equal(t, "\x1b[2A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1A\r", <-out)
	assert.Equal(t, "Are you sure?: No\n", <-out)
	assert.Equal(t, false, <-res)
}

func TestDownAndUpCase(t *testing.T) {
	in, out, res := confirmForTest(t)
	assert.Equal(t,
		"\rAre you sure? + Yes\n\r              - No\n\r",
		<-out)
	in <- "\x1b\x5b\x42" // down
	<-out                // frame render
	in <- "\x1b\x5b\x41" // up
	<-out                // frame render
	in <- "\x0d"         // enter
	<-out                // clear
	assert.Equal(t, "Are you sure?: Yes\n", <-out)
	assert.Equal(t, true, <-res)
}

func overflowForTest(t *testing.T) (in, out chan string, result chan string) {
	t.Helper()
	cio, opts := testIOforDropdown(t, 12, 4)
	result = make(chan string)
	go func() {
		defer close(result)
		v, err := Dropdown("Pick letter", []string{
			"A", "B", "C", "D", "E",
		}, opts)
		assert.NoError(t, err)
		result <- v
	}()

	return cio.In, cio.Out, result
}

func TestMoreItems(t *testing.T) {
	in, out, res := overflowForTest(t)
	assert.Equal(t, "\rPick letter \n\r+ A\n\r- B\n\r~ 3 of 5 more\n\r", <-out)
	in <- "\x1b\x5b\x42" // down
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r+ B\n\r- C\n\r~ 2 of 5 more\n\r",
		<-out)
	in <- "\x1b\x5b\x42" // down
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r+ C\n\r- D\n\r~ 1 of 5 more\n\r",
		<-out)
	in <- "\x1b\x5b\x42" // down
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r+ D\n\r- E\n\r\n\r",
		<-out)
	in <- "\x1b\x5b\x42" // down
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r- D\n\r+ E\n\r\n\r",
		<-out)
	in <- "\x1b\x5b\x42" // down, no more items, might bell
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r- D\n\r+ E\n\r\n\r",
		<-out)
	in <- "\x1b\x5b\x41" // up
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r+ D\n\r- E\n\r\n\r",
		<-out)
	in <- "\x0d" // enter
	assert.Equal(t, "\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r", <-out)
	assert.Equal(t, "Pick letter: D\n", <-out)
	assert.Equal(t, "D", <-res)
}

func TestMoreItemsUp(t *testing.T) {
	in, out, res := overflowForTest(t)
	assert.Equal(t, "\rPick letter \n\r+ A\n\r- B\n\r~ 3 of 5 more\n\r", <-out)
	in <- "\x1b\x5b\x42" // down
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r+ B\n\r- C\n\r~ 2 of 5 more\n\r",
		<-out)
	in <- "\x1b\x5b\x41" // up
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rPick letter \n\r+ A\n\r- B\n\r~ 3 of 5 more\n\r",
		<-out)
	in <- "\x0d" // enter
	assert.Equal(t, "\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r", <-out)
	assert.Equal(t, "Pick letter: A\n", <-out)
	assert.Equal(t, "A", <-res)
}

func otherDropdownForTest(t *testing.T) (in, out chan string, result chan string) {
	t.Helper()
	cio, opts := testIOforDropdown(t, 12, 4)
	result = make(chan string)
	go func() {
		defer close(result)
		v, err := Dropdown("Neque porro", []string{
			"Lorem ipsum",
			"dolor sit amet",
			"adipiscing elit",
			"Quisque porttitor",
			"condimentum libero",
		}, opts)
		assert.NoError(t, err)
		result <- v
	}()

	return cio.In, cio.Out, result
}

func TestDropdownFiltering(t *testing.T) {
	in, out, res := otherDropdownForTest(t)
	assert.Equal(t, "\rNeque porro \n\r+ Lorem ip…\n\r- dolor si…\n\r~ 3 of 5 more\n\r", <-out)
	in <- "c"
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rNeque porro \n\r+ condimen…\n\r",
		<-out)
	in <- "\x7f" // backspace
	assert.Equal(t,
		"\x1b[2A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1A\r\rNeque porro \n\r+ Lorem ip…\n\r- dolor si…\n\r~ 3 of 5 more\n\r",
		<-out)
	in <- "l"
	assert.Equal(t,
		"\x1b[4A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[3A\r\rNeque porro \n\r+ Lorem ip…\n\r- condimen…\n\r",
		<-out)
	in <- "\x1b\x5b\x42" // down
	assert.Equal(t,
		"\x1b[3A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[2A\r\rNeque porro \n\r- Lorem ip…\n\r+ condimen…\n\r",
		<-out)
	in <- "\x0d" // enter
	assert.Equal(t, "\x1b[3A\r\x1b[K\x1b[1B\r\x1b[K\x1b[1B\r\x1b[K\x1b[2A\r", <-out)
	assert.Equal(t, "Neque porro: condimentum libero\n", <-out)
	assert.Equal(t, "condimentum libero", <-res)
}
