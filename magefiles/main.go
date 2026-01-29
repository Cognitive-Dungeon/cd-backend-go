package main

import (
	cb "cognitive-builder"

	"github.com/magefile/mage/mg"
)

// Default target
var Default = Menu

func Menu() {
	// 1. Глобальные задачи (которые не методы)
	globals := map[string]func(){
		"Build All": All,
		"Clean":     Clean,
	}

	// 2. Запуск меню
	// Просто перечисляем структуры. Рефлексия сделает остальное.
	// Если добавите Tools{}, его методы (Build, Run) появятся сами.
	cb.InteractiveMenu(globals, Server{}, Inspector{}, Assets{})
}

func All() {
	mg.SerialDeps(Assets.CookAll, Server.Build, Inspector.Build)
	cb.Stats.PrintSummary()
	cb.Stats.WriteManifest()
}

func Clean() {
	cb.CleanAll()
}
