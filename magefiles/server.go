package main

import (
	"path/filepath"

	cb "cognitive-builder"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Server mg.Namespace

const (
	serverRoot = "cognitive-server"
	serverMod  = "cognitive-server"
)

// Build собирает основной сервер
func (ns Server) Build() error {
	return cb.Build(ns).
		WithSrcDir(filepath.Join(serverRoot, "cmd", "server")).
		WithMeta().
		WithModuleName(serverMod).
		Exec()
}

// Run запускает сервер
func (ns Server) Run() error {
	mg.Deps(Server.Build) // Автоматически пропустится, если ничего не менялось
	cb.LogHeader("Starting Cognitive Server")
	return sh.RunV(cb.ExePath(cb.NamespaceToString(ns)))
}

func (ns Server) Clean() error {
	cb.LogHeader("Cleaning artifacts")
	return sh.Rm(cb.ExePath(cb.NamespaceToString(ns)))
}

// Rebuild выполняет чистую пересборку сервера
func (ns Server) Rebuild() {
	// SerialDeps гарантирует, что Clean выполнится перед Build
	mg.SerialDeps(ns.Clean, ns.Build)
}

type Inspector mg.Namespace

func (ns Inspector) Build() error {
	return cb.Build(ns).
		WithSrcDir(filepath.Join(serverRoot, "cmd", "inspector")).
		Exec()
}

func (ns Inspector) Run() error {
	mg.Deps(Inspector.Build)
	cb.LogHeader("Starting Map Inspector")
	return sh.RunV(cb.ExePath(cb.NamespaceToString(ns)))
}

func (ns Inspector) Clean() error {
	cb.LogHeader("Cleaning artifacts")
	return sh.Rm(cb.ExePath(cb.NamespaceToString(ns)))
}

// Rebuild выполняет чистую пересборку инспектора
func (ns Inspector) Rebuild() {
	mg.SerialDeps(ns.Clean, ns.Build)
}
