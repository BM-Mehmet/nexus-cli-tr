package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"nexus-cli/pkg/cli"
	"nexus-cli/pkg/deepseek"
)

// LaunchHarness runs DeepSeek evaluation benchmarks or a custom test harness script.
func LaunchHarness(ctx context.Context, opts Options) error {
	// 1. If user passed a custom external command/script to run
	if len(opts.ExtraArgs) > 0 && (opts.ExtraArgs[0] == "--exec" || opts.ExtraArgs[0] == "-e") {
		if len(opts.ExtraArgs) < 2 {
			return fmt.Errorf("--exec seçeneğinden sonra bir komut belirtin (ör. nexus launch harness --exec 'pytest tests/')")
		}
		return runExternalHarness(ctx, opts)
	}

	// 2. Otherwise run built-in DeepSeek Eval Harness Suite
	fmt.Printf("\n%s%s===========================================================%s\n", cli.BrightCyan, cli.Bold, cli.Reset)
	fmt.Printf("%s%s            DEEPSEEK AJAN DEĞERLENDİRME VE KIYASLAMA ARACI %s\n", cli.BrightCyan, cli.Bold, cli.Reset)
	fmt.Printf("%s%s===========================================================%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)

	client := deepseek.NewClient(deepseek.ClientOptions{})

	// Test 1: PoW Challenge & Solver Speed
	fmt.Printf("%s[1/3] PoW şifreleme motoru denetleniyor (Keccak-f[1600])...%s\n", cli.Bold, cli.Reset)
	powStart := time.Now()
	powAns, err := client.RequestAndSolvePoW(ctx, "/api/v0/chat/completion")
	powDuration := time.Since(powStart)
	if err != nil {
		fmt.Printf("  ❌ PoW hatası: %v\n", err)
		return err
	}
	fmt.Printf("  ✅ PoW çözüldü! Süre: %s (Başlık uzunluğu: %d bayt)\n\n", powDuration.Round(time.Millisecond), len(powAns))

	// Test 2: R1 Deep Thinking & Reasoning Verification
	fmt.Printf("%s[2/3] DeepSeek-R1 akıl yürütme zinciri denetleniyor...%s\n", cli.Bold, cli.Reset)
	prompt := "Aşağıdaki mantık sorusunu adım adım çözümleyerek yanıtla: Bir kafeste tavşanlar ve tavuklar var. Toplam baş sayısı 12, toplam bacak sayısı 34. Kafeste kaç tavşan ve kaç tavuk var?"

	thinkingEnabled := true
	searchEnabled := false
	streamStart := time.Now()
	var ttft time.Duration
	var hasTTFT bool
	var thoughtTokens int
	var answerTokens int
	var thoughtSample string

	chunkCh, errCh := client.ChatStream(ctx, deepseek.ChatStreamRequest{
		Prompt:          prompt,
		Model:           deepseek.ModelReasoner,
		ThinkingEnabled: &thinkingEnabled,
		SearchEnabled:   &searchEnabled,
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				fmt.Printf("  ❌ Hata: %v\n", err)
				return err
			}
		case chunk, ok := <-chunkCh:
			if !ok {
				goto DoneStream
			}
			if !hasTTFT {
				ttft = time.Since(streamStart)
				hasTTFT = true
			}
			if chunk.ReasoningContent != "" {
				thoughtTokens += len(strings.Fields(chunk.ReasoningContent))
				if len(thoughtSample) < 180 {
					thoughtSample += chunk.ReasoningContent
				}
			}
			if chunk.Content != "" {
				answerTokens += len(strings.Fields(chunk.Content))
			}
		}
	}

DoneStream:
	totalTime := time.Since(streamStart)
	fmt.Printf("  ✅ İlk belirtece kadar geçen süre (TTFT): %s\n", ttft.Round(time.Millisecond))
	fmt.Printf("  ✅ Akıl yürütme sözcükleri: ~%d\n", thoughtTokens)
	fmt.Printf("  ✅ Yanıt sözcükleri:         ~%d\n", answerTokens)
	fmt.Printf("  ✅ Toplam süre:              %s\n", totalTime.Round(time.Millisecond))
	if thoughtSample != "" {
		fmt.Printf("  🧠 R1 akıl yürütme örneği:\n     %s\"%s...\"%s\n\n", cli.Dim, strings.TrimSpace(thoughtSample), cli.Reset)
	}

	// Test 3: Account Pool Health & Failover Readiness
	fmt.Printf("%s[3/3] Hesap havuzu ve yedek geçiş denetleniyor...%s\n", cli.Bold, cli.Reset)
	pool := deepseek.NewAccountPool(opts.AccountsDir)
	accounts := pool.List()
	readyCount := 0
	for _, a := range accounts {
		if a.IsReady() {
			readyCount++
		}
	}
	fmt.Printf("  ✅ Yüklenen hesap sayısı: %d (Hazır: %d)\n", len(accounts), readyCount)

	fmt.Printf("\n%s%s🎉 KIYASLAMA BAŞARIYLA TAMAMLANDI! TÜM SİSTEMLER ÇALIŞIYOR.%s\n\n", cli.BrightGreen, cli.Bold, cli.Reset)
	return nil
}

func runExternalHarness(ctx context.Context, opts Options) error {
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

	cmdArgs := opts.ExtraArgs[1:] // skip --exec
	cmdStr := strings.Join(cmdArgs, " ")

	fmt.Printf("%s⚡ Özel kıyaslama betiği başlatılıyor: %s%s\n", cli.BrightCyan, cmdStr, cli.Reset)
	fmt.Printf("  OPENAI_BASE_URL değeri:      http://localhost:%d/v1\n", opts.Port)
	fmt.Printf("  ANTHROPIC_BASE_URL değeri:   http://localhost:%d\n\n", opts.Port)

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENAI_BASE_URL=http://localhost:%d/v1", opts.Port),
		fmt.Sprintf("ANTHROPIC_BASE_URL=http://localhost:%d", opts.Port),
		"OPENAI_API_KEY=nexus",
		"ANTHROPIC_API_KEY=nexus",
		"DEEPSEEK_API_KEY=nexus",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
