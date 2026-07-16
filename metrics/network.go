package metrics

import (
	"fmt"
	"net"
)

// GetNetworkInterfaces возвращает список сетевых адаптеров с их IP-адресами (для п.9.1 ТЗ).
// Эта функция может вызываться из модуля диагностики.
func GetNetworkInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения сетевых интерфейсов: %v", err)
	}

	var result []string
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var ips []string
		for _, addr := range addrs {
			// Отфильтровываем пустые или loopback адреса по необходимости
			ips = append(ips, addr.String())
		}

		if len(ips) > 0 {
			result = append(result, fmt.Sprintf("%s: %v", iface.Name, ips))
		}
	}

	return result, nil
}
