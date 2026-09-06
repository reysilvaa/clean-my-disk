package service

import "clean-my-disk/internal/repository"

const TaskName = "CleanMyDisk"

func InstallWeeklySchedule(executable string) error {
	return repository.RegisterWeeklyTask(TaskName, executable)
}

func RemoveSchedule() error {
	return repository.RemoveTask(TaskName)
}
