# Скрипт диагностики с GUI (WinForms) (п.9.4 ТЗ)

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$exePath = Join-Path $PSScriptRoot "..\Venera.exe"

$form = New-Object System.Windows.Forms.Form
$form.Text = "Venera Diagnostics"
$form.Size = New-Object System.Drawing.Size(600, 400)
$form.StartPosition = "CenterScreen"

$textBox = New-Object System.Windows.Forms.TextBox
$textBox.Multiline = $true
$textBox.ScrollBars = "Vertical"
$textBox.Dock = "Fill"
$textBox.Font = New-Object System.Drawing.Font("Consolas", 10)
$textBox.ReadOnly = $true
$textBox.BackColor = [System.Drawing.Color]::Black
$textBox.ForeColor = [System.Drawing.Color]::Lime

$button = New-Object System.Windows.Forms.Button
$button.Text = "Запустить диагностику"
$button.Dock = "Bottom"
$button.Height = 40

$button.Add_Click({
    $textBox.Text = "Запуск диагностики, пожалуйста подождите...`r`n"
    $form.Refresh()

    if (-Not (Test-Path $exePath)) {
        $textBox.Text += "ОШИБКА: Venera.exe не найден!`r`n"
        return
    }

    # Запускаем exe и перехватываем вывод
    $pinfo = New-Object System.Diagnostics.ProcessStartInfo
    $pinfo.FileName = $exePath
    $pinfo.Arguments = "--diagnose"
    $pinfo.RedirectStandardOutput = $true
    $pinfo.RedirectStandardError = $true
    $pinfo.UseShellExecute = $false
    $pinfo.CreateNoWindow = $true

    $p = New-Object System.Diagnostics.Process
    $p.StartInfo = $pinfo
    $p.Start() | Out-Null
    $p.WaitForExit()

    $stdout = $p.StandardOutput.ReadToEnd()
    $stderr = $p.StandardError.ReadToEnd()

    $textBox.Text += $stdout
    if ($stderr) {
        $textBox.Text += "`r`nОШИБКИ:`r`n" + $stderr
    }
    
    $textBox.Text += "`r`nГотово. Архив диагностики сохранен в папке проекта."
})

$form.Controls.Add($textBox)
$form.Controls.Add($button)

[void]$form.ShowDialog()
