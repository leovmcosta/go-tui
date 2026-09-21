// Copyright 2026 Serge Smertin
// SPDX-License-Identifier: MIT

package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"text/template"
)

type password struct {
	config

	Label string
	Hide  bool

	LabelTemplate   string
	labelTemplate   *template.Template
	AnswerTemplate  string
	answerTemplate  *template.Template
	ReplacementChar rune
	typed           []rune

	CheckFn func(rawPassword string) (string, bool)

	// TODO: special case for testing?..
	makeTermIO func(in io.Reader, out io.Writer) (*termIO, error)
}

func newPassword() *password {
	return &password{
		config: config{
			ctx: context.Background(),
			out: os.Stderr,
			in:  os.Stdin,
		},
		Label:           "Enter password:",
		LabelTemplate:   DefaultLabelTemplate,
		AnswerTemplate:  DefaultAnswerTemplate,
		ReplacementChar: '*',
		makeTermIO:      makeTermIO,
	}
}

func Password(label string, option ...opt) (string, error) {
	p := newPassword()
	p.Label = label
	err := opts(option).Apply(p)
	if err != nil {
		return "", err
	}
	err = p.parseTemplates()
	if err != nil {
		return "", err
	}

	return p.run()
}

//nolint:cyclop,errcheck // TODO: expand input to support replacement chars and remove this
func (p *password) run() (string, error) {
	io, err := p.makeTermIO(p.in, p.out)
	if err != nil {
		return "", err
	}
	defer io.Restore()
	var frame bytes.Buffer
	var init bool
	for {
		if !init {
			init = true
		}
		err = p.labelTemplate.Execute(&frame, p.Label)
		if err != nil {
			return "", fmt.Errorf("label: %w", err)
		}
		for range len(p.typed) {
			frame.WriteRune(p.ReplacementChar)
		}
		frame.WriteRune('\r')
		frame.WriteRune('\n')
		_, err = frame.WriteTo(io.out)
		if err != nil {
			return "", err
		}
		select {
		case <-p.ctx.Done():
			return "", p.ctx.Err()
		default:
			key, err := io.ReadKey()
			io.clear(1, &frame)
			if err != nil {
				if errors.Is(err, ErrUnknownRune) {
					continue
				}
				frame.WriteTo(io) // clear the screen
				// Ctrl+C or Ctrl+D
				return "", err
			}
			switch key {
			case keyEnter:
				frame.WriteTo(io)

				return string(p.typed), nil
			case 0x7f: // backspace
				if len(p.typed) > 0 {
					fmt.Fprintf(&frame, "\x1b[1K")
					p.typed = p.typed[:len(p.typed)-1]
				}
			default:
				p.typed = append(p.typed, key)
			}
		}
	}
}

func (p *password) parseTemplates() (err error) {
	tmpl := template.New("dropdown").Funcs(colorFns)
	p.labelTemplate, err = tmpl.New("label").Parse(mustEndWith(p.LabelTemplate, ' '))
	if err != nil {
		return fmt.Errorf("label: %w", err)
	}
	p.answerTemplate, err = tmpl.New("answer").Parse(mustEndWith(p.AnswerTemplate, '\n'))
	if err != nil {
		return fmt.Errorf("answer: %w", err)
	}

	return nil
}
