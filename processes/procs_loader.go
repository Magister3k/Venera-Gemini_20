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
	procsMap  = make(map[string]models.ProcCfg)
	procsMu   sync.RWMutex
	procsPath string
)

func InitProcFile() {
	// Определение пути к processes.toml относительно исполняемого файла
	// Это важно при запуске в виде службы.
	exePath, err := os.Executable()
	if err == nil {
		procsPath = filepath.Join(filepath.Dir(exePath), "processes.toml")
	} else {
		// В случае ошибки
		procsPath = "processes.toml"
	}
}

// LoadProcs загружает список процессов из файла в формате TOML (processes.toml).
func LoadProcs() error {
	procsMu.Lock()
	defer procsMu.Unlock()

	procsMap = make(map[string]models.ProcCfg)

	if _, err := os.Stat(procsPath); os.IsNotExist(err) {
		// Файла нет - нормальная ситуация
		return nil
	}

	var pf models.ProcsFile
	_, err := toml.DecodeFile(procsPath, &pf)
	if err != nil {
		return fmt.Errorf("ошибка парсинга processes.toml: %v", err)
	}

	for k, v := range pf.Procs {
		v.ID = k                             // Убеждаемся, что ID совпадает с ключом
		v.Status = models.StatusStopped      // При загрузке все остановлены
		procsMap[k] = v
	}

	return nil
}

// SaveProcs сохраняет список процессов в файл processes.toml
func SaveProcs() error {
	procsMu.RLock()
	defer procsMu.RUnlock()

	pf := models.ProcsFile{
		Procs: make(map[string]models.ProcCfg),
	}

	for k, v := range procsMap {
		pf.Procs[k] = v
	}

	// Для надежности создадим резервную копию (отказоустойчивость)
	if _, err := os.Stat(procsPath); err == nil {
		_ = os.Rename(procsPath, procsPath+".bak")
	}

	file, err := os.Create(procsPath)
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

// GetProc возвращает конфигурацию процесса по ID
func GetProc(id string) (models.ProcCfg, bool) {
	procsMu.RLock()
	defer procsMu.RUnlock()
	p, ok := procsMap[id]
	return p, ok
}

// GetAllProcs возвращает список всех процессов
func GetAllProcs() []models.ProcCfg {
	procsMu.RLock()
	defer procsMu.RUnlock()
	var list []models.ProcCfg
	for _, p := range procsMap {
		list = append(list, p)
	}
	return list
}

// AddOrUpdProc добавляет или обновляет процесс и сохраняет в файл
func AddOrUpdProc(p models.ProcCfg) error {
	procsMu.Lock()
	if p.ID == "" {
		procsMu.Unlock()
		return fmt.Errorf("ID процесса не может быть пустым")
	}
	if p.Status == "" {
		p.Status = models.StatusStopped
	}
	procsMap[p.ID] = p
	procsMu.Unlock()
	return SaveProcs()
}

// DelProc удаляет процесс и сохраняет изменения
func DelProc(id string) error {
	procsMu.Lock()
	delete(procsMap, id)
	procsMu.Unlock()
	return SaveProcs()
}

// UpdProcStatus обновляет только статус процесса (в памяти)
func UpdProcStatus(id string, status models.ProcStatus) {
	procsMu.Lock()
	defer procsMu.Unlock()
	if p, ok := procsMap[id]; ok {
		p.Status = status
		procsMap[id] = p
	}
}

