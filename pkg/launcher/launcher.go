package launcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"nexus-cli/pkg/cli"
	"nexus-cli/pkg/deepseek"
	"nexus-cli/pkg/server"
)

// Options holds common launcher options.
type Options struct {
	Port        int
	Model       string
	AccountsDir string
	ExtraArgs   []string
}

// EnsureServer verifies if Nexus proxy server is running, or starts an embedded one.
// Returns the server instance, whether it was started by the launcher, and any error.
func EnsureServer(ctx context.Context, port int, accountsDir string) (*server.Server, bool, error) {
	if port <= 0 {
		port = 9655
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	client := http.Client{Timeout: 350 * time.Millisecond}
	resp, err := client.Get(healthURL)
	if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
		_ = resp.Body.Close()
		return nil, false, nil // Server already running
	}

	fmt.Printf("%s[Nexus]%s Yerel DeepSeek API sunucusu arka planda başlatılıyor (Bağlantı noktası: %d)...\n", cli.BrightCyan, cli.Reset, port)

	srv := server.NewServer(server.ServerOptions{
		Host:        "127.0.0.1",
		Port:        port,
		AccountsDir: accountsDir,
	})

	serverErrCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
	}()

	// Wait for server to become healthy
	start := time.Now()
	for time.Since(start) < 3*time.Second {
		select {
		case err := <-serverErrCh:
			return nil, false, fmt.Errorf("sunucu başlatılamadı: %w", err)
		default:
		}

		resp, err := client.Get(healthURL)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			fmt.Printf("%s[Nexus]%s ✅ Yerel proxy etkin: http://localhost:%d\n\n", cli.BrightGreen, cli.Reset, port)
			return srv, true, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return srv, true, nil
}

// LaunchClaude runs Claude Code CLI wired to Nexus proxy.
func LaunchClaude(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	modelName := opts.Model
	if modelName == "" {
		modelName = deepseek.ModelReasoner
	}
	modelCfg := deepseek.ResolveModelConfig(modelName)
	modelName = modelCfg.ID

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Printf("\n%s❌ 'claude' (Claude Code) CLI sistemde bulunamadı.%s\n", cli.Red, cli.Reset)
		fmt.Println("Kurmak için şu komutu çalıştırın:")
		fmt.Printf("  %snpm install -g @anthropic-ai/claude-code%s\n\n", cli.BrightCyan, cli.Reset)
		return fmt.Errorf("claude çalıştırılabilir dosyası PATH içinde bulunamadı")
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

	var args []string
	hasModelArg := false
	hasNameArg := false

	for _, a := range opts.ExtraArgs {
		if a == "--model" || a == "-m" {
			hasModelArg = true
		}
		if a == "--name" || a == "-n" {
			hasNameArg = true
		}
		args = append(args, a)
	}

	if !hasModelArg {
		args = append([]string{"--model", modelName}, args...)
	}
	if !hasNameArg {
		args = append([]string{"-n", fmt.Sprintf("DeepSeek (%s)", modelName)}, args...)
	}

	fmt.Printf("%s⚡ Claude Code başlatılıyor (%s)...%s\n", cli.BrightGreen, modelCfg.RealModel, cli.Reset)
	fmt.Printf("  API temel adresi: http://localhost:%d\n", opts.Port)
	fmt.Printf("  Etkin model:  %s\n", modelName)
	fmt.Printf("  Modeli değiştirme: Claude içinde /model <ad> veya nexus launch claude -m <ad>\n\n")

	cmd := exec.CommandContext(ctx, claudePath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ANTHROPIC_BASE_URL=http://localhost:%d", opts.Port),
		"ANTHROPIC_AUTH_TOKEN=nexus",
		"ANTHROPIC_API_KEY=nexus",
		fmt.Sprintf("ANTHROPIC_MODEL=%s", modelName),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// LaunchCodex runs OpenAI Codex / OpenCode / Aider CLI wired to Nexus proxy.
func LaunchCodex(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	// Check codex, opencode, or aider
	var binPath, toolName string
	candidates := []string{"codex", "opencode", "aider"}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			binPath = p
			toolName = c
			break
		}
	}

	if binPath == "" {
		fmt.Printf("\n%s❌ Codex / OpenCode / Aider CLI bulunamadı.%s\n", cli.Yellow, cli.Reset)
		fmt.Println("Önerilen kurulum seçenekleri:")
		fmt.Printf("  1. Aider:    %spip install aider-chat%s\n", cli.BrightCyan, cli.Reset)
		fmt.Printf("  2. OpenCode: %snpm install -g opencode-ai%s\n\n", cli.BrightCyan, cli.Reset)
		fmt.Println("Ortam değişkenlerini elle de ayarlayabilirsiniz:")
		fmt.Printf("  export OPENAI_BASE_URL=\"http://localhost:%d/v1\"\n", opts.Port)
		fmt.Println("  export OPENAI_API_KEY=\"nexus\"")
		return fmt.Errorf("codex/opencode çalıştırılabilir dosyası bulunamadı")
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

	fmt.Printf("%s⚡ %s başlatılıyor (DeepSeek OpenAI API üzerinden)...%s\n", cli.BrightGreen, toolName, cli.Reset)
	fmt.Printf("  OpenAI temel adresi: http://localhost:%d/v1\n\n", opts.Port)

	cmd := exec.CommandContext(ctx, binPath, opts.ExtraArgs...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
		"OPENAI_API_KEY=nexus",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// LaunchChatGPT opens ChatGPT Desktop app and prints bridge connection guide.
func LaunchChatGPT(ctx context.Context, opts Options) error {
	if opts.Port <= 0 {
		opts.Port = 9655
	}

	srv, started, err := EnsureServer(ctx, opts.Port, opts.AccountsDir)
	if err != nil {
		return err
	}
	if started && srv != nil {
		// Keep server alive while user interacts
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
	}

	fmt.Printf("%s%s=== CHATGPT DESKTOP VE YEREL KÖPRÜ ===%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)

	if runtime.GOOS == "darwin" {
		appPath := "/Applications/ChatGPT.app"
		if _, err := os.Stat(appPath); err == nil {
			fmt.Printf("✅ ChatGPT Desktop uygulaması açılıyor...\n")
			_ = exec.Command("open", "-a", appPath).Run()
		} else {
			fmt.Println("ℹ️ ChatGPT Desktop uygulaması /Applications içinde bulunamadı.")
		}
	}

	fmt.Println("\nYerel DeepSeek köprüsü ayarları:")
	fmt.Printf("  • OpenAI API adresi: %shttp://localhost:%d/v1%s\n", cli.BrightGreen, opts.Port, cli.Reset)
	fmt.Printf("  • API anahtarı:    %snexus%s (veya istediğiniz bir değer)\n", cli.BrightGreen, cli.Reset)
	fmt.Printf("  • Önerilen model:  %sdeepseek-reasoner%s (R1) veya %sdeepseek-chat%s (V3)\n", cli.Bold, cli.Reset, cli.Bold, cli.Reset)
	fmt.Printf("\nBunu ChatGPT veya üçüncü taraf uygulamanızın temel URL ayarına girin.\n")
	fmt.Printf("Çıkmak için Ctrl+C tuşlarına basın...\n")

	<-ctx.Done()
	return nil
}

// PrintLaunchMenu outputs the launcher help menu.
func PrintLaunchMenu(w io.Writer) {
	fmt.Fprintf(w, `
%s%sNexus Evrensel Başlatıcı (Tümleştirmeler)%s

Kullanım: nexus launch <platform> [ek seçenekler]

Kullanılabilir platformlar:
  %sclaude%s          Claude Code CLI (Anthropic Messages API /v1/messages)
  %sdsh%s             DeepSeek Harness (@deepseek-ai/dsh CLI)
  %scodex%s           OpenAI Codex CLI (OpenAI Responses wire API)
  %sopencode%s        OpenCode CLI (otomatik yapılandırma)
  %sgoose%s           Block Goose AI Agent (OpenAI API sağlayıcısı)
  %saider%s           Aider eşli programlama CLI (DeepSeek R1)
  %schatgpt%s         ChatGPT Desktop ve yerel OpenAI köprüsü
  %sharness%s         Nexus yerleşik ajan değerlendirme aracı
  %spool%s            Çoklu hesap havuzu izleyicisi

Örnekler:
  nexus launch claude
  nexus launch claude -p "Bu projeyi incele"
  nexus launch dsh
  nexus launch codex
  nexus launch opencode
  nexus launch goose
  nexus launch aider
  nexus launch chatgpt
  nexus launch pool
`, cli.BrightCyan, cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset,
		cli.Bold, cli.Reset)
}
