package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"nexus-cli/pkg/cli"
)

// LaunchDSH launches the official DeepSeek Harness agent (@deepseek-ai/dsh).
func LaunchDSH(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("kullanıcı ana dizini bulunamadı: %w", err)
	}

	// 1. Prepare configuration directory ~/.nexus/launch/dsh
	dshDir := filepath.Join(home, ".nexus", "launch", "dsh")
	if err := os.MkdirAll(dshDir, 0o755); err != nil {
		return fmt.Errorf("dsh yapılandırma dizini oluşturulamadı: %w", err)
	}

	settingsPath := filepath.Join(dshDir, "settings.yaml")
	patchPath := filepath.Join(dshDir, "patch.yaml")

	settingsContent := fmt.Sprintf(`agent-default-model:
  provider: nexus
  model: deepseek-reasoner
llm-pi-ai:
  providers:
    nexus:
      displayName: "Nexus (DeepSeek R1/V3)"
      apiKeyEnv: NEXUS_API_KEY
      api: openai-completions
      baseURL: "http://localhost:%d/v1"
      models:
        - name: deepseek-reasoner
          displayName: "DeepSeek R1 (Reasoner)"
        - name: deepseek-chat
          displayName: "DeepSeek V3 (Chat)"
`, opts.Port)

	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0o644); err != nil {
		return fmt.Errorf("dsh ayarları yazılamadı: %w", err)
	}

	patchContent := fmt.Sprintf(`- id: settings
  config:
    path: %q
`, settingsPath)

	if err := os.WriteFile(patchPath, []byte(patchContent), 0o644); err != nil {
		return fmt.Errorf("dsh yama dosyası yazılamadı: %w", err)
	}

	// 2. Ensure local Nexus API server is running
	srv, started, err := EnsureServer(ctx, opts.Port, opts.AccountsDir)
	if err != nil {
		return err
	}
	if started && srv != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
	}

	// 3. Find dsh executable or fallback to npx
	var cmd *exec.Cmd
	launchArgs := append([]string{"web", "--patch", patchPath}, opts.ExtraArgs...)

	if dshBin, err := exec.LookPath("dsh"); err == nil {
		cmd = exec.CommandContext(ctx, dshBin, launchArgs...)
	} else if npxBin, err := exec.LookPath("npx"); err == nil {
		npxArgs := append([]string{"-y", "@deepseek-ai/dsh@latest", "web", "--patch", patchPath}, opts.ExtraArgs...)
		cmd = exec.CommandContext(ctx, npxBin, npxArgs...)
	} else {
		fmt.Printf("\n%s❌ DeepSeek Harness (dsh) veya Node.js (npx) bulunamadı.%s\n", cli.Red, cli.Reset)
		fmt.Println("Kurulum adımları:")
		fmt.Println("  1. Node.js'i kurun: https://nodejs.org/")
		fmt.Printf("  2. dsh'yi kurun:    %snpm install -g @deepseek-ai/dsh@latest%s\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("dsh veya npx çalıştırılabilir dosyası PATH içinde bulunamadı")
	}

	fmt.Printf("%s⚡ DeepSeek Harness (dsh) başlatılıyor...%s\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  Nexus API adresi: http://localhost:%d/v1\n", opts.Port)
	fmt.Printf("  Model:            deepseek-reasoner\n")
	fmt.Printf("  Yama dosyası:  %s\n\n", patchPath)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"NEXUS_API_KEY=nexus",
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
		"OPENAI_API_KEY=nexus",
	)

	return cmd.Run()
}
