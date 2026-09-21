// Copyright 2026 Serge Smertin
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type clipboard struct{}

func ShouldPasteFromClipboard() string {
	content, _ := (&clipboard{}).Read() //nolint:errcheck // ignore clipboard errors

	return content
}

var clipboardPasteImplementations = map[string][][]string{
	"darwin": {
		{"pbpaste"},
	},
	"linux": {
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
		{"wl-paste"},
	},
	"windows": {
		{"powershell", "-command", "Get-Clipboard"},
	},
}

func (cr *clipboard) pasteCommand() (*exec.Cmd, error) {
	impls, ok := clipboardPasteImplementations[runtime.GOOS]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, runtime.GOOS)
	}
	for _, args := range impls {
		_, err := exec.LookPath(args[0])
		if err != nil {
			continue
		}

		return exec.Command(args[0], args[1:]...), nil
	}

	return nil, fmt.Errorf("%w: no clipboard paste utility found", ErrUnsupportedPlatform)
}

func (cr *clipboard) Read() (string, error) {
	cmd, err := cr.pasteCommand()
	if err != nil {
		return "", fmt.Errorf("paste command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("run: %w", err)
	}
	content := strings.TrimRight(string(output), "\n\r")

	return content, nil
}
