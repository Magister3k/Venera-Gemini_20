package utils

import (
	"fmt"
	"os/exec"
)

// ShowBalloonNotify отображает системное всплывающее уведомление Windows (Balloon)
func ShowBalloonNotify(title, message string) {
	psCmd := fmt.Sprintf(`
		[void][System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms");
		$notification = New-Object System.Windows.Forms.NotifyIcon;
		$notification.Icon = [System.Drawing.SystemIcons]::Information;
		$notification.BalloonTipIcon = "Info";
		$notification.BalloonTipTitle = "%s";
		$notification.BalloonTipText = "%s";
		$notification.Visible = $true;
		$notification.ShowBalloonTip(5000);
	`, title, message)

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psCmd)
	_ = cmd.Start() // Асинхронный запуск
}
