package service

import (
	"clean-my-disk/internal/repository"
)

func IsElevated() bool {
	return repository.IsAdmin()
}

func RelaunchElevated(executable string, args []string) error {
	return repository.RunElevated(executable, args)
}
