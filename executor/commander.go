package executor

import (
	"os/exec"
)

//go:generate mockgen -package=executor -source=./commander.go -destination=./commander_mock.go
type commander interface {
	execGoRun(targetModFilePath, targetFilePath string) (cmdOut []byte, err error)
	execGoListAll() (cmdOut []byte, err error)
	execGoListMod() (cmdOut []byte, err error)
	execGoModTidy(targetDir string) (err error)
}

type defaultCommander struct{}

func newDefaultCommander() *defaultCommander {
	return &defaultCommander{}
}

func (dc *defaultCommander) execGoRun(targetModFilePath, targetFilePath string) (cmdOut []byte, err error) {
	cmd := exec.Command("go", "run", "-modfile="+targetModFilePath, targetFilePath)
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

func (dc *defaultCommander) execGoListMod() (cmdOut []byte, err error) {
	cmd := exec.Command("go", "list", "-m")
	cmdOut, cmdErr := cmd.Output()
	if cmdErr != nil {
		return nil, cmdErr
	}
	return cmdOut, nil
}

func (dc *defaultCommander) execGoModTidy(targetDir string) (err error) {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = targetDir
	if cmdErr := cmd.Run(); cmdErr != nil {
		return cmdErr
	}
	return nil
}
