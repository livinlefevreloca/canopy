package sso

import (
	"bytes"
	"os/exec"
)

type SSOLoginError struct {
	Message string
}

func (e *SSOLoginError) Error() string {
	return e.Message
}

func ExecAwsSSOLogin(profile string) error {
	cmd := exec.Command("aws", "sso", "login", "--profile", profile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	stderrData := string(stderr.Bytes())
	if err != nil {
		return &SSOLoginError{
			Message: "Failed to execute aws sso login command: " + err.Error() + "\n" + stderrData,
		}
	}

	if cmd.ProcessState.ExitCode() != 0 {
		return &SSOLoginError{
			Message: "AWS SSO login command failed with exit code " + string(cmd.ProcessState.ExitCode()) + "\n" + stderrData,
		}
	}

	return nil
}
