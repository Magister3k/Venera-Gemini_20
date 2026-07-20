package services

import (
	"fmt"
	"os"
)

// Из-за ограничений кросс-компиляции и встроенных инструментов Go,
// внедрение иконки в PE (Portable Executable) файл Windows (exe)
// обычно осуществляется на этапе сборки через утилиты `rsrc` или `go-winres`.
// Наиболее правильный подход в Go для добавление иконки -
// это генерация syso файла, который линкуется при сборке `go build`.

// GenerateSysoResource создает файл ресурса (syso) с иконкой и манифестом для сборки
// Этот модуль является утилитой, выполняемой перед сборкой (go generate).
func GenerateSysoResource(iconPath, manifestPath, outSysoPath string) error {
	// Проверка наличия иконки
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		return fmt.Errorf("иконка не найдена по пути: %s", iconPath)
	}

	// Мы предоставляем интерфейс,
	// который может быть вызван утилитой сборки (build.ps1), написанной на Go,
	// или как генератор (go:generate).

	return fmt.Errorf("внедрение иконки должно осуществляться инструментом go-winres в скрипте build.ps1. Модуль готов к расширению.")
}

// ExtractIcon извлекает иконку из памяти/ассетов (например, для systray)
func ExtractIcon(targetPath string) error {
	// Заглушка: здесь можно вставить []byte иконки (go:embed),
	// если она нужна для отрисовки трея.
	// В реальной системе используется go:embed
	return nil
}
