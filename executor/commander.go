package executor

import "os/exec"

//go:generate mockgen -package=executor -source=./commander.go -destination=./commander_mock.go
type commander interface {
	execGoRun(targetFile string) (cmdOut []byte, err error)
	execGoListAll() (cmdOut []byte, err error)
}

type defaultCommander struct{}

func newDefaultCommander() *defaultCommander {
	return &defaultCommander{}
}

func (dc *defaultCommander) execGoRun(targetFile string) (cmdOut []byte, err error) {
	cmd := exec.Command("go", "run", targetFile)
	cmdOut, cmdErr := cmd.Output()
	if cmdErr != nil {
		return nil, cmdErr
	}
	return cmdOut, nil
}

func (dc *defaultCommander) execGoListAll() (cmdOut []byte, err error) {
	cmd := exec.Command("go", "list", "./...")
	cmdOut, cmdErr := cmd.Output()
	if cmdErr != nil {
		return nil, cmdErr
	}
	return cmdOut, nil
}
