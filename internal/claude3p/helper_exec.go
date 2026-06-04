package claude3p

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func runHelperJSON(helperPath string) ([]byte, error) {
	cmd := exec.Command(helperPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s: %s", helperPath, msg)
	}
	return bytes.TrimSpace(stdout.Bytes()), nil
}
