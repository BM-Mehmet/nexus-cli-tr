package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"nexus-cli/pkg/cli"
)

// LaunchAider runs Aider pair-programming CLI wired to Nexus proxy.
func LaunchAider(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	aiderBin, err := exec.LookPath("aider")
	if err != nil {
		fmt.Printf("\n%s❌ 'aider' CLI sistemde bulunamadı.%s\n", cli.Red, cli.Reset)
		fmt.Println("Kurmak için:")
		fmt.Printf("  %spip install aider-chat%s\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("aider çalıştırılabilir dosyası PATH içinde bulunamadı")
	}

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

	fmt.Printf("%s⚡ Aider eşli programlama CLI başlatılıyor (DeepSeek R1 / V3)...%s\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  OpenAI temel URL: http://localhost:%d/v1\n", opts.Port)
	fmt.Printf("  Model:            openai/deepseek-reasoner\n\n")

	args := []string{"--model", "openai/deepseek-reasoner"}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, aiderBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENAI_API_BASE=http://localhost:%d/v1", opts.Port),
		"OPENAI_API_KEY=nexus",
	)

	return cmd.Run()
}
