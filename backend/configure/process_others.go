//go:build !windows

//go:dev !windows

package configure

import (
	"os/exec"
)

// 在非 Windows 平台不修改子程序設定
func ConfigureChildProcess(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
