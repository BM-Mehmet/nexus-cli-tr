package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"nexus-cli/pkg/deepseek"
)
// REPL manages interactive terminal chat sessions.
type REPL struct {
	client          *deepseek.Client
	model           string
	thinkingEnabled bool
	searchEnabled   bool
}

// NewREPL initializes a REPL instance.
func NewREPL(client *deepseek.Client, defaultModel string) *REPL {
	if defaultModel == "" {
		defaultModel = deepseek.ModelReasoner
	}

	cfg := deepseek.ResolveModelConfig(defaultModel)
	return &REPL{
		client:          client,
		model:           cfg.ID,
		thinkingEnabled: cfg.ThinkingEnabled,
		searchEnabled:   cfg.SearchEnabled,
	}
}

// Run starts the interactive chat loop.
func (r *REPL) Run(ctx context.Context) error {
	PrintBanner(os.Stdout)

	profile, err := r.client.Auth().ValidateToken(ctx)
	if err == nil && profile != nil {
		userLabel := profile.Nickname
		if userLabel == "" {
			userLabel = profile.Email
		}
		if userLabel == "" {
			userLabel = profile.Phone
		}
		fmt.Printf("%s Kullanıcı:%s %s%s%s\n", Gray, Reset, BrightGreen, userLabel, Reset)
	} else {
		fmt.Printf("%s Uyarı: Belirteç doğrulanamadı. Yenilemek için 'nexus auth login' komutunu kullanın.%s\n", Yellow, Reset)
	}

	fmt.Printf("%s Model:%s %s%s%s (Akıl yürütme: %s, Arama: %s)\n",
		Gray, Reset, BrightCyan, r.model, Reset,
		boolLabel(r.thinkingEnabled), boolLabel(r.searchEnabled),
	)
	fmt.Printf("%s Yardım için %s/help%s, çıkmak için %s/exit%s yazın.%s\n\n", Gray, Bold, Reset, Bold, Reset, Gray)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("%s%snexus%s %s❯%s ", BrightCyan, Bold, Reset, Yellow, Reset)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			if r.handleCommand(ctx, input) {
				break
			}
			continue
		}

		r.ask(ctx, input)
		fmt.Println()
	}

	fmt.Printf("\n%sGörüşmek üzere!%s\n", Gray, Reset)
	return nil
}

func (r *REPL) handleCommand(ctx context.Context, cmdLine string) bool {
	parts := strings.Fields(cmdLine)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/exit", "/quit", "/q":
		return true

	case "/clear", "/cls":
		fmt.Print("\033[H\033[2J")
		return false

	case "/new", "/reset":
		r.client.ResetSession()
		fmt.Printf("%s Yeni sohbet oturumu başladı.%s\n\n", Green, Reset)
		return false

	case "/whoami", "/user":
		profile, err := r.client.Auth().ValidateToken(ctx)
		if err != nil {
			fmt.Printf("%s Hata: %v%s\n\n", Red, err, Reset)
			return false
		}
		fmt.Printf("%s Hesap: ID=%s, Takma ad=%s, E-posta=%s, Telefon=%s%s\n\n",
			BrightCyan, profile.ID, profile.Nickname, profile.Email, profile.Phone, Reset)
		return false

	case "/model", "/m":
		if len(parts) < 2 {
			fmt.Printf("%s Geçerli model: %s%s\n\n", Gray, r.model, Reset)
			return false
		}
		target := parts[1]
		switch strings.ToLower(target) {
		case "r1", "reasoner":
			target = deepseek.ModelReasoner
		case "v3", "chat":
			target = deepseek.ModelChat
		case "r1-search":
			target = deepseek.ModelReasonerSearch
		case "v3-search":
			target = deepseek.ModelChatSearch
		}
		cfg := deepseek.ResolveModelConfig(target)
		r.model = cfg.ID
		r.thinkingEnabled = cfg.ThinkingEnabled
		r.searchEnabled = cfg.SearchEnabled
		fmt.Printf("%s Model değiştirildi: %s%s\n\n", BrightGreen, r.model, Reset)
		return false

	case "/thinking", "/t":
		if len(parts) >= 2 {
			r.thinkingEnabled = parseBoolArg(parts[1])
		} else {
			r.thinkingEnabled = !r.thinkingEnabled
		}
		fmt.Printf("%s Akıl yürütme: %s%s\n\n", Gray, boolLabel(r.thinkingEnabled), Reset)
		return false

	case "/search", "/s":
		if len(parts) >= 2 {
			r.searchEnabled = parseBoolArg(parts[1])
		} else {
			r.searchEnabled = !r.searchEnabled
		}
		fmt.Printf("%s Web araması: %s%s\n\n", Gray, boolLabel(r.searchEnabled), Reset)
		return false

	case "/help", "/h", "/?":
		fmt.Println(`Komutlar:
  /help                  - Bu yardım ekranını göster
  /new                   - Yeni sohbet başlat (oturumu sıfırla)
  /model <ad>            - Modeli seç (r1, v3, deepseek-reasoner, deepseek-chat)
  /thinking [on|off]     - Akıl yürütmeyi aç veya kapat
  /search [on|off]       - Web aramasını aç veya kapat
  /whoami                - Hesap bilgilerini göster
  /clear                 - Ekranı temizle
  /exit                  - Programdan çık`)
		fmt.Println()
		return false

	default:
		fmt.Printf("%s Bilinmeyen komut: %s. Komutları görmek için /help yazın.%s\n\n", Yellow, cmd, Reset)
		return false
	}
}

func (r *REPL) ask(ctx context.Context, prompt string) bool {
	spinner := NewSpinner("DeepSeek PoW hesaplanıyor ve sunucuya bağlanılıyor...")
	spinner.Start()

	chunkCh, errCh := r.client.ChatStream(ctx, deepseek.ChatStreamRequest{
		Prompt:          prompt,
		Model:           r.model,
		ThinkingEnabled: &r.thinkingEnabled,
		SearchEnabled:   &r.searchEnabled,
	})

	hasStarted := false
	inThinking := false
	hasThinkingHeader := false
	hasContentHeader := false

	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				spinner.Stop()
				fmt.Printf("\n%s❌ Hata: %v%s\n", Red, err, Reset)
				return false
			}
		case chunk, ok := <-chunkCh:
			if !ok {
				spinner.Stop()
				if inThinking {
					fmt.Print("\n\n")
				}
				select {
				case err := <-errCh:
					if err != nil {
						fmt.Printf("\n%s❌ Hata: %v%s\n", Red, err, Reset)
						return false
					}
				default:
				}
				return true
			}

			if !hasStarted {
				spinner.Stop()
				hasStarted = true
			}

			// Handle streaming reasoning
			if chunk.ReasoningContent != "" {
				if !hasThinkingHeader {
					fmt.Printf("\n%s💭 Akıl yürütme:%s\n%s%s", Cyan, Reset, Gray, Italic)
					hasThinkingHeader = true
					inThinking = true
				}
				fmt.Print(chunk.ReasoningContent)
			}

			// Handle streaming content
			if chunk.Content != "" {
				if inThinking {
					fmt.Printf("%s\n\n", Reset)
					inThinking = false
				}
				if !hasContentHeader {
					fmt.Printf("%s🤖 DeepSeek (%s):%s\n", BrightGreen, r.model, Reset)
					hasContentHeader = true
				}
				fmt.Print(chunk.Content)
			}
		}
	}
}

func boolLabel(b bool) string {
	if b {
		return fmt.Sprintf("%saçık%s", BrightGreen, Reset)
	}
	return fmt.Sprintf("%skapalı%s", Gray, Reset)
}

func parseBoolArg(s string) bool {
	s = strings.ToLower(s)
	return s == "1" || s == "true" || s == "yes" || s == "on" || s == "ha"
}
