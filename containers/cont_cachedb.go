package containers

import (
	"os/exec"
	"strings"

	"venera/logging"
	"venera/utils"
)

// startCacheDb поднимает контейнер с кэширующей СУБД
func startCacheDb(pathPodman, pathImage string) {

	// Запуск Podman
	ok := startPodman(pathPodman)
	if ok {
		// Запуск контейнера
		container := "cachedb"
		ok = startContainer(pathPodman, container)
		if !ok {
			imageStrArr := []string{"dragonfly", "redis"}
			// Установка и запуск контейнера
			ok = setupContainer(pathPodman, pathImage, container, imageStrArr)
		}
	}
	if !ok {
		utils.ShowBalloonNotification("Venera", "Ошибка запуска кэширующей СУБД")
	}
}

// startContainer Запускает контейнер
func startContainer(podman, container  string) (started bool){
	out, _ := exec.Command(podman, "ps", "-a", "--format", "{{.Names}}").Output()
	if strings.Contains(string(out), container) {
		// Запуск контейнера
		if err := exec.Command(podman, "start", container).Run(); err != nil {
			logging.Log.Errorf("Ошибка запуска контейнера %s: %v", container, err)
			return
		}
	} else {
		logging.Log.Warnf("Контейнер %s не найден", container)
		return
	}
	started = true
	return
}

// setupContainer устанавливает и запускает контейнер
func setupContainer(podman, image, container string, imageStrArr []string) (started bool) {
		// Загрузка образа из файла
		if err := exec.Command(podman, "load", "-i", image).Run(); err != nil {
			logging.Log.Errorf("Ошибка загрузки образа из файла %s: %v", image, err)
			return
		}
		for _, imageStr := range imageStrArr {
			// Получение имени загруженного образа
			out, _ := exec.Command(podman, "image", "inspect", imageStr, "--format", "{{index .NamesHistory 0}}").Output()
			if len(out) > 1 {
				imageName := strings.TrimSpace(string(out))
				// Создание и запуск контейнера
				err := exec.Command(podman, "run", "-d", "--name", container, "-p", "6379:6379", imageName).Run()
				if err != nil {
					logging.Log.Errorf("Ошибка создания контейнера %s: %v", container, err)
					return
				}
				break
			}
		}
		started = true
		return
}
