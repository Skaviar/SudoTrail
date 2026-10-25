// Build a self-contained static site for Cloudflare Pages or any static host.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	out := flag.String("out", "dist", "static output directory")
	flag.Parse()
	if err := build(*out); err != nil {
		log.Fatal(err)
	}
}
func build(out string) error {
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	for _, name := range []string{"index.html", "styles.css", "app.js", "photo.js", "engine.js", "solver-worker.js"} {
		content, err := os.ReadFile(filepath.Join("web", name))
		if err != nil {
			return err
		}
		if name == "index.html" {
			content = bytes.Replace(content, []byte(`data-engine="api"`), []byte(`data-engine="wasm"`), 1)
		}
		if err := os.WriteFile(filepath.Join(out, name), content, 0644); err != nil {
			return err
		}
	}
	support, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "wasm_exec.js"), support, 0644); err != nil {
		return err
	}
	command := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", filepath.Join(out, "solver.wasm"), "./cmd/wasm")
	command.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "_headers"), []byte("/solver.wasm\n  Content-Type: application/wasm\n  Cache-Control: public, max-age=0, must-revalidate\n/wasm_exec.js\n  Cache-Control: public, max-age=0, must-revalidate\n"), 0644); err != nil {
		return err
	}
	info, err := os.Stat(filepath.Join(out, "solver.wasm"))
	if err != nil {
		return err
	}
	if info.Size() >= 25*1024*1024 {
		return fmt.Errorf("WASM exceeds Cloudflare Pages' 25 MiB asset limit")
	}
	fmt.Printf("Static site ready in %s (Go WASM: %.2f MiB, 1,000 embedded puzzles).\n", out, float64(info.Size())/(1024*1024))
	return nil
}
