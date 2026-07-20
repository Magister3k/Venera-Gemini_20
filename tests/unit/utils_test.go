package unit_test

import (
	"testing"
	"venera/models"
	"venera/utils"
)

// TestIDGenerator проверяет модуль генерации идентификаторов
// Цель: убедиться, что базовые утилиты (модуль utils) работают предсказуемо
// и не зависят от состояния системы.
func TestIDGenerator(t *testing.T) {
	t.Log("Запуск юнит-теста модуля утилит")

	id := utils.GenerateID()
	if len(id) != 16 {
		t.Errorf("Ожидалась длина ID = 16, получено %d", len(id))
	}

	id2 := utils.GenerateID()
	if id == id2 {
		t.Errorf("Сгенерированные ID не должны совпадать")
	}
}

// TestModelsInitialization проверяет правильность инициализации констант
// и базовых структур без внешней зависимости от конфигов.
func TestModelsInitialization(t *testing.T) {
	if models.StatusRunning != "running" {
		t.Errorf("Ожидался статус running")
	}
	if models.SourceNetwork != "network" {
		t.Errorf("Ожидался тип источника network")
	}
}
