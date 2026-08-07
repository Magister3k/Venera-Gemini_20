package containers

import (
	"os/exec"

	"venera/logging"
)

// startPodman запускает Podman
func startPodman(podman string) (started bool) {

	// Проверка наличия исполняемого файла
 	if _, err := exec.LookPath(podman); err != nil {
 		logging.Log.Warnf("Podman не найден по пути: %s", podman)		
		return
 	}

	// Запуск виртуальной машины
	if err := exec.Command(podman, "machine", "start").Run(); err != nil {
		logging.Log.Errorf("Ошибка запуска Podman: %v", err)
		return
	}

	started = true
	return
}
