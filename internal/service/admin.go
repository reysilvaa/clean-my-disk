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

func ShowToast(title, message string) error {
	return repository.ShowToast(title, message)
}

func EnsureToastRegistration(executable string) error {
	return repository.EnsureToastRegistration(executable)
}
