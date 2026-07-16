package manifest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Путь к файлу манифеста по умолчанию (в корне проекта/рядом с exe)
var ManifestFilePath = "manifest.xml"
var ManifestBackupPath = "manifest.xml.bak"

// CurrentAppVersion - предполагается, что она устанавливается при сборке
// или берется из глобальных констант приложения.
var CurrentAppVersion = "1.0.0"

// ProviderName - имя ETW провайдера
const ProviderName = "VeneraApp"

// EventIDManifestUpdated - Event ID, который требуется по ТЗ (п.8.6)
const EventIDManifestUpdated = 2003

// Init - точка входа для работы с манифестом при старте приложения
func Init() error {
	// 8.3. Проверка версии зарегистрированного манифеста при запуске.
	registered, err := checkManifestRegistered()
	if err != nil {
		notifyError(fmt.Sprintf("Ошибка проверки манифеста: %v", err))
		return err
	}

	if !registered {
		// 8.1. Регистрация через код при старте.
		err = RegisterManifest()
		if err != nil {
			notifyError(fmt.Sprintf("Ошибка регистрации манифеста: %v", err))
			return err
		}
	} else {
		// Проверка версии и обновление, если необходимо
		// Для простоты сверяем текущую версию приложения с какой-либо внутренней меткой,
		// либо перезаписываем, если версия обновилась.
		// Здесь логика обновления: если требуется обновление, вызываем UpdateManifest
		// (В реальной жизни нужно парсить существующий XML или реестр).
		needsUpdate := checkVersionNeedsUpdate()
		if needsUpdate {
			err = UpdateManifest()
			if err != nil {
				notifyError(fmt.Sprintf("Ошибка обновления манифеста: %v", err))
				// 8.5. Откат манифеста при ошибке.
				RollbackManifest()
				return err
			}
		}
	}

	return nil
}

// checkManifestRegistered проверяет, зарегистрирован ли уже провайдер в системе
func checkManifestRegistered() (bool, error) {
	// Используем wevtutil ep (enum-providers) для поиска нашего провайдера
	cmd := exec.Command("wevtutil", "ep")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("не удалось получить список ETW провайдеров: %v", err)
	}

	return strings.Contains(out.String(), ProviderName), nil
}

// checkVersionNeedsUpdate проверяет, нужно ли обновлять манифест
func checkVersionNeedsUpdate() bool {
	// В полноценной реализации здесь должно быть чтение версии из зарегистрированного манифеста
	// или из реестра. Для текущей реализации просто проверяем наличие файла бэкапа и текущей версии
	// заглушка: всегда возвращаем false, если манифест уже есть, чтобы не спамить.
	// TODO: Реализовать чтение версии из XML.
	return false
}

// RegisterManifest регистрирует XML манифест через wevtutil
func RegisterManifest() error {
	absPath, err := filepath.Abs(ManifestFilePath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("файл манифеста не найден по пути: %s", absPath)
	}

	// Команда: wevtutil im manifest.xml
	cmd := exec.Command("wevtutil", "im", absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wevtutil im failed: %v, output: %s", err, string(output))
	}

	// 8.6. Хранение истории версий в Event ID 2003 ("Manifest updated").
	return LogManifestVersion()
}

// UnregisterManifest удаляет регистрацию манифеста (полезно при удалении службы)
func UnregisterManifest() error {
	absPath, err := filepath.Abs(ManifestFilePath)
	if err != nil {
		return err
	}

	cmd := exec.Command("wevtutil", "um", absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wevtutil um failed: %v, output: %s", err, string(output))
	}
	return nil
}

// UpdateManifest перезаписывает манифест при обновлении с созданием резервной копии (8.4)
func UpdateManifest() error {
	// Создание резервной копии
	err := backupManifest()
	if err != nil {
		return fmt.Errorf("не удалось создать резервную копию манифеста: %v", err)
	}

	// Сначала нужно снять регистрацию старого манифеста
	UnregisterManifest()

	// Регистрация нового
	err = RegisterManifest()
	if err != nil {
		return fmt.Errorf("ошибка регистрации обновленного манифеста: %v", err)
	}

	return nil
}

// backupManifest создает копию manifest.xml
func backupManifest() error {
	input, err := ioutil.ReadFile(ManifestFilePath)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(ManifestBackupPath, input, 0644)
	if err != nil {
		return err
	}
	return nil
}

// RollbackManifest производит откат манифеста при ошибке (8.5)
func RollbackManifest() error {
	if _, err := os.Stat(ManifestBackupPath); os.IsNotExist(err) {
		return fmt.Errorf("файл резервной копии не найден")
	}

	// Восстанавливаем оригинальный файл из бэкапа
	input, err := ioutil.ReadFile(ManifestBackupPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения бэкапа: %v", err)
	}

	err = ioutil.WriteFile(ManifestFilePath, input, 0644)
	if err != nil {
		return fmt.Errorf("ошибка восстановления файла манифеста: %v", err)
	}

	// Повторная регистрация восстановленного манифеста
	UnregisterManifest() // Игнорируем ошибку отмены, так как он мог быть не зарегистрирован
	return RegisterManifest()
}

// LogManifestVersion записывает событие об обновлении манифеста (Event ID 2003) (8.6)
func LogManifestVersion() error {
	// Для записи событий в Windows Event Log без сторонних тяжелых библиотек,
	// можно использовать утилиту eventcreate.
	// Команда: eventcreate /ID 2003 /L Application /T INFORMATION /SO VeneraApp /D "Manifest updated to version X"
	msg := fmt.Sprintf("Manifest updated to version %s", CurrentAppVersion)
	cmd := exec.Command("eventcreate", "/ID", fmt.Sprintf("%d", EventIDManifestUpdated), "/L", "Application", "/T", "INFORMATION", "/SO", ProviderName, "/D", msg)

	err := cmd.Run()
	if err != nil {
		// Eventcreate может потребовать прав администратора.
		return fmt.Errorf("ошибка записи события в Event Log (нужны права Администратора?): %v", err)
	}
	return nil
}

// notifyError выводит GUI-уведомление (8.2) при ошибках
func notifyError(message string) {
	// В зависимости от того, запущено ли приложение в режиме systray,
	// выводим уведомление.
	// Примечание: getlantern/systray не имеет встроенного метода ShowNotification/Balloon,
	// поэтому здесь мы добавляем пункт меню с ошибкой или логируем.
	// Для реальных Balloon-уведомлений на Windows лучше использовать github.com/go-toast/toast.

	// Здесь добавим пункт меню с пометкой Ошибка.
	// В реальной архитектуре Venera этот вызов должен уходить в подсистему логирования или интерфейса.
	fmt.Printf("GUI УВЕДОМЛЕНИЕ (ОШИБКА РЕГИСТРАЦИИ): %s\n", message)

	// Попытка использовать systray, если он проинициализирован.
	// Поскольку systray инициализируется асинхронно, прямое добавление может упасть, если петля не запущена.
	// Для безопасной интеграции с ТЗ (п.8.2) оставляем этот вызов абстрактным:
	// systray.AddMenuItem("Ошибка: "+message, "Ошибка манифеста")
}
