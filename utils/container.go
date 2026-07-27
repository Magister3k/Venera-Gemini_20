package utils

import (
	"os/exec"
	"strings"

	"venera/logging"
	"venera/models"
	"venera/tray"
)

// setupCacheDbContainer проверяет и поднимает контейнер с кэширующей СУБД
func setupCacheDbContainer(paths models.PathsConfig) {
	podman := paths.PodmanExe

	// Проверка наличия podman
 	if _, err := exec.LookPath(podman); err != nil {
 		logging.Log.Warnf("Podman не найден по пути: %s", podman)
 		tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
		return
 	}

	// Запуск podman machine (если требуется на Windows)
	if _, err := exec.Command(podman, "machine", "start").Run(); err != nil {
		logging.Log.Errorf("Ошибка запуска Podman: %v", err)
		tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
		return
	}

	// Проверка наличия контейнера CacheDb
	out, _ := exec.Command(podman, "ps", "-a", "--format", "{{.Names}}").Output()
	if strings.Contains(string(out), "cachedb") {
		// Запуск контейнера
		if _, err := exec.Command(podman, "start", "cachedb").Run(); err != nil {
			logging.Log.Errorf("Ошибка запуска контейнера: %v", err)
			tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
		}
	} else {
		logging.Log.Warnf("Контейнер cachedb не найден")

		// Загрузка образа кэширующей СУБД из файла
		if _, err := exec.Command(podman, "load", "-i", paths.DbImage).Run(); err != nil {
			logging.Log.Errorf("Ошибка загрузки образа из файла %s: %v", paths.DbImage, err)
			tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
			return
		}
		imageStrArr := []string{"dragonfly", "redis"}
		for _, imageStr := range imageStrArr {
			// Получение имени загруженного образа
			out, _ := exec.Command(podman, "image", "inspect", imageStr, "--format", "{{index .NamesHistory 0}}").Output()
			if len(out) > 1 {
				imageName := strings.TrimSpace(string(out))
				// Создание и запуск контейнера
				err := exec.Command(podman, "run", "-d", "--name", "cachedb", "-p", "6379:6379", imageName).Run()
				if err != nil {
					logging.Log.Errorf("Ошибка создания контейнера: %v", err)
					tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
				}
				break
			}
		}
	}
}
