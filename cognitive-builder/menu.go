package cognitive_builder

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/pterm/pterm"
)

// MenuEntry связывает название пункта меню и функцию
type MenuEntry struct {
	Label  string
	Action func() error
}

// InteractiveMenu строит меню на основе переданных неймспейсов и функций
// globalFunctions: map["Name"]func()
// namespaces: список структур (например: Server{}, Inspector{})
func InteractiveMenu(globalFunctions map[string]func(), namespaces ...interface{}) {
	LogHeader("Cognitive Builder")

	var entries []MenuEntry

	// 1. Глобальные функции
	for name, fn := range globalFunctions {
		wrapper := func() error {
			fn()
			return nil
		}
		entries = append(entries, MenuEntry{Label: name, Action: wrapper})
	}

	// 2. Namespace методы
	for _, ns := range namespaces {
		val := reflect.ValueOf(ns)
		typ := reflect.TypeOf(ns)
		nsName := typ.Name()

		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			methodVal := val.Method(i)

			// приватные пропускаем
			if method.PkgPath != "" {
				continue
			}

			label := fmt.Sprintf("%s: %s", nsName, method.Name)

			action := func() error {
				results := methodVal.Call(nil)
				if len(results) > 0 {
					if err, ok := results[0].Interface().(error); ok {
						return err
					}
				}
				return nil
			}

			entries = append(entries, MenuEntry{Label: label, Action: action})
		}
	}

	// 3. Сортировка
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Label < entries[j].Label
	})

	// 4. Exit в конец
	entries = append(entries, MenuEntry{
		Label: "Exit",
		Action: func() error {
			return nil
		},
	})

	// 5. Нумерация (красиво выравниваем)
	width := len(strconv.Itoa(len(entries)))
	var options []string
	entryMap := make(map[string]func() error)

	for i, e := range entries {
		num := fmt.Sprintf("%*d.", width, i)
		label := fmt.Sprintf("%s %s", num, e.Label)

		options = append(options, label)
		entryMap[label] = e.Action
	}

	// 6. Показ меню
	selected, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithDefaultText("Select task to run").
		WithMaxHeight(15).
		Show()

	// 7. Запуск
	if action, exists := entryMap[selected]; exists {
		if err := action(); err != nil {
			pterm.Error.Println("Task failed:", err)
			Notify("Task Failed", selected, true)
		}
	}
}
