// Package ssa предоставляет функции для построения SSA представления
package ssa

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Builder отвечает за построение SSA из исходного кода Go
type Builder struct {
	fset *token.FileSet
}

// NewBuilder создаёт новый экземпляр Builder
func NewBuilder() *Builder {
	return &Builder{
		fset: token.NewFileSet(),
	}
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// ParseAndBuildSSA парсит исходный код Go и создаёт SSA представление
// Возвращает SSA функцию по имени
func (b *Builder) ParseAndBuildSSA(source string, funcName string) (*ssa.Function, error) {
	if _, err := parser.ParseFile(b.fset, "source.go", source, parser.ParseComments); err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "ssa-builder-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "source.go")
	if err := os.WriteFile(tmpFile, []byte(source), 0o644); err != nil {
		return nil, fmt.Errorf("cannot write temp source file: %w", err)
	}

	cfg := &packages.Config{
		Mode: packages.LoadSyntax,
		Fset: b.fset,
	}
	pkgs, err := packages.Load(cfg, "file="+tmpFile)
	if err != nil {
		return nil, fmt.Errorf("package load failed: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found")
	}
	if len(pkgs[0].Errors) > 0 {
		return nil, fmt.Errorf("errors while loading packages: %v", pkgs[0].Errors)
	}

	prog, ssaPkgs := ssautil.AllPackages(pkgs, ssa.BuilderMode(0))
	prog.Build()

	for _, ssaPkg := range ssaPkgs {
		if fn := ssaPkg.Func(funcName); fn != nil {
			return fn, nil
		}
	}

	return nil, fmt.Errorf("function %q not found", funcName)
}

func (b *Builder) PrintSSABlocks(fn *ssa.Function) {
	fmt.Printf("Function: %s\n", fn.String())
	fmt.Printf("Total blocks: %d\n", len(fn.Blocks))

	for _, block := range fn.Blocks {
		fmt.Printf("\nBlock #%d (Index=%d):\n", block.Index, block.Index)

		fmt.Println("  Body:")
		if len(block.Instrs) == 0 {
			fmt.Println("    <none>")
		} else {
			for i, instr := range block.Instrs {
				fmt.Printf("    %2d: %s\n", i, instr.String())
			}
		}

		fmt.Print("  Progenitors: ")
		if len(block.Preds) == 0 {
			fmt.Print("<none>")
		} else {
			for i, pred := range block.Preds {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%d", pred.Index)
			}
		}
		fmt.Println()

		fmt.Print("  Descendants: ")
		if len(block.Succs) == 0 {
			fmt.Print("<none>")
		} else {
			for i, succ := range block.Succs {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%d", succ.Index)
			}
		}
		fmt.Println()
	}
}
