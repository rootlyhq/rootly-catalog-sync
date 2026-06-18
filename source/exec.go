package source

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type ExecSource struct {
	Command string
	Args    []string
}

func NewExecSource(command string, args []string) *ExecSource {
	return &ExecSource{Command: command, Args: args}
}

func (s *ExecSource) Name() string { return "exec" }

func (s *ExecSource) Load(ctx context.Context) ([]Entry, error) {
	cmd := exec.CommandContext(ctx, s.Command, s.Args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec %s failed: %w: %s", s.Command, err, stderr.String())
	}

	return Parse(stdout.Bytes())
}
