package utils

import (
	"os"

	"golang.org/x/sys/windows"
)

// IsAdmin проверяет, запущено ли приложение с правами Администратора (п.16 ТЗ)
func IsAdmin() bool {
	// Для Windows мы пытаемся открыть физический диск (PhysicalDrive) на чтение
	// или просто проверяем встроенную функцию из golang.org/x/sys/windows.
	// Ограничимся простой проверкой открытия токена доступа.
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		// Fallback: попытаемся открыть системный диск, доступный только админам
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}
	return member
}
