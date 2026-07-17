package processes

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
	"venera/models"
)

var (
	processesMap = make(map[string]models.ProcessConfig)
	processesMu  sync.RWMutex
	ProcsPath    string
)

func init() {
	// Определение пути к processes.toml относительно исполняемого файла
	// Это важно при запуске в виде службы.
	exePath, err := os.Executable()
	if err == nil {
		ProcsPath = filepath.Join(filepath.Dir(exePath), "processes.toml")
	} else {
		// Fallback
		ProcsPath = "processes.toml"
	}
}

// LoadProcesses загружает список процессов из файла processes.toml (п.1.9 ТЗ).
func LoadProcesses() error {
	processesMu.Lock()
	defer processesMu.Unlock()

	processesMap = make(map[string]models.ProcessConfig)

	if _, err := os.Stat(ProcsPath); os.IsNotExist(err) {
		// Файла нет - нормальная ситуация
		return nil
	}

	var pf models.ProcessesFile
	_, err := toml.DecodeFile(ProcsPath, &pf)
	if err != nil {
		return fmt.Errorf("ошибка парсинга processes.toml: %v", err)
	}

	for k, v := range pf.Processes {
		v.ID = k                             // Убеждаемся, что ID совпадает с ключом
		v.Status = models.StatusStopped      // При загрузке все остановлены
		processesMap[k] = v
	}

	return nil
}

// SaveProcesses сохраняет список процессов в файл processes.toml
func SaveProcesses() error {
	processesMu.RLock()
	defer processesMu.RUnlock()

	pf := models.ProcessesFile{
		Processes: make(map[string]models.ProcessConfig),
	}

	for k, v := range processesMap {
		pf.Processes[k] = v
	}

	// Для надежности создадим резервную копию (отказоустойчивость)
	if _, err := os.Stat(ProcsPath); err == nil {
		_ = os.Rename(ProcsPath, ProcsPath+".bak")
	}

	file, err := os.Create(ProcsPath)
	if err != nil {
		return fmt.Errorf("ошибка создания processes.toml: %v", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(pf); err != nil {
		return fmt.Errorf("ошибка кодирования processes.toml: %v", err)
	}

	return nil
}

// GetProcess возвращает конфигурацию процесса по ID
func GetProcess(id string) (models.ProcessConfig, bool) {
	processesMu.RLock()
	defer processesMu.RUnlock()
	p, ok := processesMap[id]
	return p, ok
}

// GetAllProcesses возвращает список всех процессов
func GetAllProcesses() []models.ProcessConfig {
	processesMu.RLock()
	defer processesMu.RUnlock()
	var list []models.ProcessConfig
	for _, p := range processesMap {
		list = append(list, p)
	}
	return list
}

// AddOrUpdateProcess добавляет или обновляет процесс и сохраняет в файл
func AddOrUpdateProcess(p models.ProcessConfig) error {
	processesMu.Lock()
	if p.ID == "" {
		processesMu.Unlock()
		return fmt.Errorf("ID процесса не может быть пустым")
	}
	if p.Status == "" {
		p.Status = models.StatusStopped
	}
	processesMap[p.ID] = p
	processesMu.Unlock()
	return SaveProcesses()
}

// RemoveProcess удаляет процесс и сохраняет изменения
func RemoveProcess(id string) error {
	processesMu.Lock()
	delete(processesMap, id)
	processesMu.Unlock()
	return SaveProcesses()
}

// UpdateProcessStatus обновляет только статус процесса (в памяти)
func UpdateProcessStatus(id string, status models.ProcessStatus) {
	processesMu.Lock()
	defer processesMu.Unlock()
	if p, ok := processesMap[id]; ok {
		p.Status = status
		processesMap[id] = p
	}
}

