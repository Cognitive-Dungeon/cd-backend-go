package cognitive_builder

import (
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
)

var (
	// Настройка логгера
	log = logrus.New()

	// Стили для заголовков
	SectionStyle = pterm.NewStyle(pterm.FgCyan, pterm.Bold)
	SuccessStyle = pterm.NewStyle(pterm.FgGreen)
	SkipStyle    = pterm.NewStyle(pterm.FgYellow)
	ErrorStyle   = pterm.NewStyle(pterm.FgRed, pterm.Bold)
)

func init() {
	// Настраиваем logrus на красивый текстовый вывод
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})
	// Pterm настройки
	pterm.ThemeDefault.SectionStyle = *SectionStyle
}

// LogHeader выводит красивый заголовок шага
func LogHeader(title string) {
	pterm.DefaultSection.Println(title)
}

// LogSuccess выводит сообщение об успехе
func LogSuccess(msg string) {
	pterm.Success.Println(msg)
}

// LogInfo выводит информационное сообщение
func LogInfo(msg string) {
	pterm.Info.Println(msg)
}

// LogSkip выводит сообщение о пропуске шага
func LogSkip(target string) {
	pterm.Info.Printfln(SkipStyle.Sprintf("⚡ Target '%s' is up to date (cached)", target))
}
