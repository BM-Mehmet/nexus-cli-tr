package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"nexus-cli/pkg/cli"
	"nexus-cli/pkg/deepseek"
	"nexus-cli/pkg/launcher"
	"nexus-cli/pkg/server"
)

const version = "1.0.0"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) < 2 {
		// If input is piped, run single prompt
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			client := deepseek.NewClient(deepseek.ClientOptions{})
			if err := cli.RunSinglePrompt(ctx, client, "", deepseek.ModelReasoner, false, false, ""); err != nil {
				fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Otherwise start interactive REPL
		client := deepseek.NewClient(deepseek.ClientOptions{})
		repl := cli.NewREPL(client, deepseek.ModelReasoner)
		if err := repl.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
		return
	}

	command := os.Args[1]

	switch command {
	case "chat":
		handleChat(ctx, os.Args[2:])

	case "serve", "server", "proxy":
		handleServe(ctx, os.Args[2:])

	case "launch":
		handleLaunch(ctx, os.Args[2:])

	case "auth":
		handleAuth(ctx, os.Args[2:])

	case "doctor":
		client := deepseek.NewClient(deepseek.ClientOptions{})
		_ = cli.RunDoctor(ctx, client)

	case "models":
		handleModels()

	case "version", "-v", "--version":
		fmt.Printf("Nexus-CLI sürümü: %s (Go için DeepSeek Web Köprüsü)\n", version)

	case "help", "-h", "--help":
		printHelp()

	default:
		// Check if user passed flags or direct prompt text
		handleRun(ctx, os.Args[1:])
	}
}

func printHelp() {
	cli.PrintBanner(os.Stdout)
	fmt.Print(`Kullanım:
  nexus [istem]                        Tek seferlik istek gönder veya sohbet başlat
  nexus chat [seçenekler]               Etkileşimli sohbeti (REPL) başlat
  nexus serve [seçenekler]              OpenAI / Anthropic uyumlu yerel API sunucusunu başlat
  nexus launch <platform>               Claude Code, Codex, ChatGPT veya Harness'i başlat
  nexus doctor                          Bağlantı ve PoW tanılamasını çalıştır
  nexus models                          Kullanılabilir DeepSeek modellerini ve özelliklerini göster
  nexus auth <komut>                    Kimlik doğrulamayı ve hesap havuzunu yönet

Başlatıcı tümleştirmeleri:
  nexus launch claude                  Claude Code CLI (DeepSeek R1 / V3 ile)
  nexus launch dsh                     DeepSeek Harness (@deepseek-ai/dsh CLI)
  nexus launch codex                   OpenAI Codex CLI
  nexus launch opencode                OpenCode CLI (otomatik yapılandırma)
  nexus launch goose                   Block Goose AI Agent
  nexus launch aider                   Aider eşli programlama CLI
  nexus launch chatgpt                 ChatGPT Desktop ve yerel köprü
  nexus launch harness                 Nexus karşılaştırma aracı
  nexus launch pool                    Çoklu hesap havuzu izleyicisi

Kimlik doğrulama komutları:
  nexus auth list                      Hesapları ve durumlarını listele
  nexus auth add <ad> [--chrome]       Hesap ekle (belirteçle veya Chrome üzerinden)
  nexus auth switch <ad>                Etkin hesabı değiştir
  nexus auth test                      Havuzdaki hesapları doğrula
  nexus auth remove <ad>                Hesabı havuzdan kaldır
  nexus auth login                     Belirteci elle gir
  nexus auth chrome                    Chrome ile oturum aç
  nexus auth whoami                    Geçerli hesap bilgilerini göster

Seçenekler:
  -m, --model <ad>         Modeli seç (varsayılan: deepseek-reasoner / R1)
  -t, --thinking           Akıl yürütmeyi etkinleştir
  -s, --search             Web aramasını etkinleştir
  --format <json>          Yapılandırılmış yanıt biçimi
  --token <belirteç>       DeepSeek belirtecini doğrudan ver
  --accounts-dir <yol>     Hesap dizini (varsayılan: ./accounts)

Örnekler:
  nexus "Merhaba, dünya!"
  nexus launch claude
  nexus launch codex
  nexus launch harness
  nexus auth list
  nexus serve --port 9655
`)
}

func handleChat(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	model := fs.String("model", deepseek.ModelReasoner, "Model adı (deepseek-reasoner, deepseek-chat)")
	fs.StringVar(model, "m", deepseek.ModelReasoner, "Model adı (kısa seçenek)")
	token := fs.String("token", "", "Belirteç")
	_ = fs.Parse(args)

	client := deepseek.NewClient(deepseek.ClientOptions{Token: *token})
	repl := cli.NewREPL(client, *model)
	if err := repl.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}
}

func handleRun(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	model := fs.String("model", deepseek.ModelReasoner, "Model adı")
	fs.StringVar(model, "m", deepseek.ModelReasoner, "Model adı (kısa seçenek)")
	thinking := fs.Bool("thinking", true, "Akıl yürütmeyi etkinleştir")
	fs.BoolVar(thinking, "t", true, "Akıl yürütmeyi etkinleştir (kısa seçenek)")
	search := fs.Bool("search", false, "Web aramasını etkinleştir")
	fs.BoolVar(search, "s", false, "Web aramasını etkinleştir (kısa seçenek)")
	format := fs.String("format", "", "Yanıt biçimi (ör. json)")
	token := fs.String("token", "", "DeepSeek belirteci")

	_ = fs.Parse(args)
	prompt := strings.Join(fs.Args(), " ")

	client := deepseek.NewClient(deepseek.ClientOptions{Token: *token})
	if err := cli.RunSinglePrompt(ctx, client, prompt, *model, *thinking, *search, *format); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}
}

func handleServe(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	host := fs.String("host", "0.0.0.0", "Dinleme adresi")
	port := fs.Int("port", 9655, "Dinleme bağlantı noktası")
	apiKey := fs.String("key", os.Getenv("PROXY_API_KEY"), "Proxy sunucusu API anahtarı (isteğe bağlı)")
	maxConcurrent := fs.Int("max-concurrent", 10, "Eşzamanlı en fazla istek sayısı")
	token := fs.String("token", "", "DeepSeek hesap belirteci")
	accountsDir := fs.String("accounts-dir", "", "Hesap dizini (varsayılan: ./accounts)")
	_ = fs.Parse(args)

	client := deepseek.NewClient(deepseek.ClientOptions{Token: *token})

	srv := server.NewServer(server.ServerOptions{
		Host:          *host,
		Port:          *port,
		ProxyAPIKey:   *apiKey,
		MaxConcurrent: *maxConcurrent,
		AccountsDir:   *accountsDir,
		Client:        client,
	})

	cli.PrintBanner(os.Stdout)
	if err := srv.StartContext(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "Sunucu hatası: %v\n", err)
		os.Exit(1)
	}
	if ctx.Err() != nil {
		fmt.Println("Sunucu durduruldu.")
	}
}

func handleAuth(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Printf("\n%s%sNexus Kimlik Doğrulama ve Çoklu Hesap Havuzu%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)
		fmt.Println("Kullanım: nexus auth <komut> [argümanlar]")
		fmt.Println("  nexus auth list                      Hesapları ve durumlarını listele")
		fmt.Println("  nexus auth add <ad> [--chrome]       Hesap ekle (belirteçle veya Chrome üzerinden)")
		fmt.Println("  nexus auth switch <ad>               Etkin hesabı değiştir")
		fmt.Println("  nexus auth test                      Havuzdaki tüm hesapları doğrula")
		fmt.Println("  nexus auth remove <ad>               Hesabı havuzdan kaldır")
		fmt.Println("  nexus auth login                     Birincil hesap belirtecini elle gir")
		fmt.Println("  nexus auth chrome                    Chrome ile otomatik oturum aç")
		fmt.Println("  nexus auth whoami                    Etkin hesap profilini göster")
		fmt.Println("  nexus auth logout                    Birincil hesap bilgilerini sil")
		return
	}

	sub := args[0]
	auth := deepseek.NewAuthManager("", "")

	switch sub {
	case "list", "ls":
		pool := deepseek.NewAccountPool("")
		accounts := pool.List()
		activeName := pool.GetActiveAccountName()

		fmt.Printf("\n%s%s=== DEEPSEEK HESAPLARI ===%s\n", cli.BrightCyan, cli.Bold, cli.Reset)
		if len(accounts) == 0 {
			fmt.Println("Henüz kayıtlı hesap yok.")
			fmt.Printf("Hesap eklemek için: %snexus auth add <ad>%s\n\n", cli.BrightCyan, cli.Reset)
			return
		}

		fmt.Printf("%-18s %-12s %-12s %-32s %s\n", "HESAP", "DURUM", "BEKLEME", "E-POSTA / PROFİL", "DOSYA")
		fmt.Println(strings.Repeat("-", 85))

		for _, acc := range accounts {
			nameTag := acc.Name
			if acc.Name == activeName || (activeName == "default" && acc.Name == "default") {
				nameTag = fmt.Sprintf("%s%s* %s%s", cli.BrightGreen, cli.Bold, acc.Name, cli.Reset)
			} else {
				nameTag = fmt.Sprintf("  %s", acc.Name)
			}

			status := fmt.Sprintf("%s✅ Hazır%s", cli.Green, cli.Reset)
			cdSec := acc.CooldownRemaining()
			cdStr := "-"
			if cdSec > 0 {
				status = fmt.Sprintf("%s⏳ Bekliyor%s", cli.Yellow, cli.Reset)
				cdStr = fmt.Sprintf("%ds", cdSec)
			} else if !acc.IsReady() {
				status = fmt.Sprintf("%s❌ Hata%s", cli.Red, cli.Reset)
			}

			userEmail := "Bilinmiyor"
			if acc.Profile != nil && acc.Profile.Email != "" {
				userEmail = acc.Profile.Email
			} else if acc.Data.Token != "" {
				t := acc.Data.Token
				if len(t) > 20 {
					userEmail = fmt.Sprintf("%s...%s", t[:8], t[len(t)-6:])
				} else {
					userEmail = t
				}
			}

			fmt.Printf("%-28s %-20s %-12s %-32s %s\n", nameTag, status, cdStr, userEmail, filepath.Base(acc.FilePath))
		}
		fmt.Printf("\n* işareti etkin birincil hesabı gösterir.\n")
		fmt.Printf("Hesabı değiştir: %snexus auth switch <ad>%s\n", cli.BrightCyan, cli.Reset)
		fmt.Printf("Hesap ekle: %snexus auth add <ad>%s\n\n", cli.BrightCyan, cli.Reset)

	case "add":
		if len(args) < 2 {
			fmt.Println("Kullanım: nexus auth add <hesap_adı> [--chrome]")
			fmt.Println("Örnek:    nexus auth add acc2")
			fmt.Println("          nexus auth add acc3 --chrome")
			return
		}
		name := args[1]
		useChrome := false
		for _, a := range args[2:] {
			if a == "--chrome" || a == "-c" {
				useChrome = true
			}
		}

		pool := deepseek.NewAccountPool("")
		if useChrome {
			fmt.Printf("Chrome üzerinden '%s' adlı yeni hesap ekleniyor...\n", name)
			helper := deepseek.NewChromeAuthHelper(9334, "", "")
			waitCtx, cancelWait := context.WithCancel(ctx)
			defer cancelWait()
			data, err := helper.LaunchAndExtract(waitCtx, func(msg string) {
				fmt.Println("[Chrome]", msg)
			})
			if err != nil || data == nil || data.Token == "" {
				fmt.Printf("❌ Chrome üzerinden alma hatası: %v\n", err)
				return
			}
			if err := pool.AddAccount(name, *data); err != nil {
				fmt.Printf("❌ Hesap kaydedilemedi: %v\n", err)
				return
			}
			fmt.Printf("✅ '%s' hesabı kaydedildi!\n", name)
		} else {
			fmt.Printf("'%s' hesabının DeepSeek userToken (veya Bearer token) değerini girin:\n> ", name)
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				tokenInput := strings.TrimSpace(scanner.Text())
				if tokenInput == "" {
					fmt.Println("❌ Belirteç girilmedi.")
					return
				}
				cleanTok := deepseek.CleanToken(tokenInput)
				fmt.Print("DeepSeek çerezlerini girin (isteğe bağlı; boş bırakmak için ENTER):\n> ")
				var cookieInput string
				if scanner.Scan() {
					cookieInput = strings.TrimSpace(scanner.Text())
				}
				data := deepseek.AuthData{
					Token:  cleanTok,
					Cookie: cookieInput,
				}
				if err := pool.AddAccount(name, data); err != nil {
					fmt.Printf("❌ Kaydetme hatası: %v\n", err)
					return
				}
				fmt.Printf("✅ '%s' hesabı kaydedildi!\n", name)
			}
		}

	case "switch", "use":
		if len(args) < 2 {
			fmt.Println("Kullanım: nexus auth switch <hesap_adı>")
			return
		}
		name := args[1]
		pool := deepseek.NewAccountPool("")
		acc, err := pool.SwitchActive(name)
		if err != nil {
			fmt.Printf("❌ Hata: %v\n", err)
			return
		}
		fmt.Printf("✅ Etkin hesap değiştirildi: %s (%s)\n", acc.Name, filepath.Base(acc.FilePath))

	case "test", "validate":
		pool := deepseek.NewAccountPool("")
		fmt.Println("Hesap havuzu doğrulanıyor...")
		pool.ValidateAll(ctx)
		for _, acc := range pool.List() {
			if acc.Profile != nil {
				name := acc.Profile.Nickname
				if name == "" {
					name = acc.Profile.Email
				}
				fmt.Printf("✅ %-14s: Bağlantı başarılı (%s, %s)\n", acc.Name, name, acc.Profile.Email)
			} else {
				fmt.Printf("❌ %-14s: Belirteç geçersiz veya ağ hatası\n", acc.Name)
			}
		}

	case "login", "set":
		fmt.Println("===========================================================")
		fmt.Println("       DEEPSEEK HESABI BAĞLAMA YÖNERGESİ       ")
		fmt.Println("===========================================================")
		fmt.Println("1. Tarayıcıda https://chat.deepseek.com adresini açın.")
		fmt.Println("2. F12 (veya Cmd+Option+I) tuşlarına basın.")
		fmt.Println("3. Application (veya Storage) bölümünde Local Storage -> chat.deepseek.com yolunu açın.")
		fmt.Println("   'userToken' değerini kopyalayın.")
		fmt.Println("===========================================================")
		fmt.Print("\nDeepSeek userToken (veya Bearer token) değerini girin: ")

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			tokenInput := strings.TrimSpace(scanner.Text())
			if tokenInput == "" {
				fmt.Println("❌ Belirteç girilmedi.")
				return
			}
			if err := auth.SetToken(tokenInput, true); err != nil {
				fmt.Printf("❌ Dosyaya kaydetme hatası: %v\n", err)
				return
			}

			fmt.Println("Doğrulanıyor...")
			profile, err := auth.ValidateToken(ctx)
			if err != nil {
				fmt.Printf("⚠️ Uyarı: Belirteç kaydedildi ancak sunucu doğrulaması başarısız: %v\n", err)
				return
			}

			name := profile.Nickname
			if name == "" {
				name = profile.Email
			}
			fmt.Printf("✅ Hesap bağlandı! Hoş geldiniz, %s (ID: %s)!\n", name, profile.ID)
		}

	case "chrome":
		fmt.Println("===========================================================")
		fmt.Println("       DEEPSEEK CHROME İLE OTOMATİK KİMLİK DOĞRULAMA     ")
		fmt.Println("===========================================================")
		fmt.Println("1. Yeni ve temiz bir Chrome penceresi açılıyor...")
		fmt.Println("2. Açılan Chrome penceresinde chat.deepseek.com adresine gidip oturum açın.")
		fmt.Println("3. Oturum açtıktan sonra DeepSeek arayüzünden kısa bir ileti gönderin (ör. 'ok').")
		fmt.Println("4. Belirteç otomatik alınır (veya burada ENTER tuşuna basın).")
		fmt.Println("===========================================================")

		helper := deepseek.NewChromeAuthHelper(9334, "", "")

		enterCh := make(chan struct{})
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				close(enterCh)
			}
		}()

		var data *deepseek.AuthData
		var err error

		waitCtx, cancelWait := context.WithCancel(ctx)
		defer cancelWait()

		go func() {
			select {
			case <-enterCh:
				fmt.Println("\n[Chrome] Belirteç doğrulanıyor...")
				d, e := helper.TryExtractNow(waitCtx)
				if e == nil && d != nil && d.Token != "" {
					data = d
					cancelWait()
				}
			case <-waitCtx.Done():
			}
		}()

		d, e := helper.LaunchAndExtract(waitCtx, func(msg string) {
			fmt.Println("[Chrome]", msg)
		})
		if data == nil {
			data = d
			err = e
		}

		if err != nil && (data == nil || data.Token == "") {
			fmt.Printf("❌ Chrome ile kimlik doğrulanamadı: %v\n", err)
			return
		}

		if err := auth.SetAuthData(*data, true); err != nil {
			fmt.Printf("❌ Kaydetme hatası: %v\n", err)
			return
		}

		fmt.Println("✅ Belirteç ve çerezler kaydedildi!")
		profile, err := auth.ValidateToken(ctx)
		if err == nil && profile != nil {
			name := profile.Nickname
			if name == "" {
				name = profile.Email
			}
			fmt.Printf("🎉 Hesap bağlandı! Hoş geldiniz, %s (ID: %s)\n", name, profile.ID)
		}

	case "whoami", "status":
		token := auth.GetToken()
		if token == "" {
			fmt.Println("❌ Belirteç bulunamadı. Eklemek için 'nexus auth login' komutunu kullanın.")
			return
		}
		profile, err := auth.ValidateToken(ctx)
		if err != nil {
			fmt.Printf("❌ Belirteç geçersiz veya süresi dolmuş: %v\n", err)
			return
		}
		fmt.Printf("✅ Bağlı hesap:\n  ID:       %s\n  Takma ad: %s\n  E-posta:  %s\n  Telefon:  %s\n",
			profile.ID, profile.Nickname, profile.Email, profile.Phone)

	case "logout", "remove", "rm":
		if len(args) >= 2 {
			name := args[1]
			pool := deepseek.NewAccountPool("")
			if err := pool.RemoveAccount(name); err != nil {
				fmt.Printf("❌ Silme hatası: %v\n", err)
				return
			}
			fmt.Printf("✅ '%s' hesabı kaldırıldı.\n", name)
			return
		}
		_ = auth.Remove()
		fmt.Println("✅ Hesap bilgileri silindi.")

	default:
		fmt.Printf("Bilinmeyen auth komutu: %s. Yardım için 'nexus auth' yazın.\n", sub)
	}
}

func handleModels() {
	fmt.Printf("%s%s=== DESTEKLENEN DEEPSEEK MODELLERİ ===%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)
	for id, cfg := range deepseek.ModelConfigs {
		reasoning := "hayır"
		if cfg.Capabilities.Reasoning {
			reasoning = "evet (R1)"
		}
		search := "hayır"
		if cfg.Capabilities.WebSearch {
			search = "evet"
		}
		fmt.Printf("  • %s%-24s%s -> %s\n", cli.Bold, id, cli.Reset, cfg.RealModel)
		fmt.Printf("    Akıl yürütme: %s | Web araması: %s\n\n", reasoning, search)
	}
}

func handleLaunch(ctx context.Context, args []string) {
	if len(args) == 0 {
		launcher.PrintLaunchMenu(os.Stdout)
		return
	}

	target := strings.ToLower(args[0])
	rawExtra := args[1:]

	model := ""
	var extra []string
	for i := 0; i < len(rawExtra); i++ {
		arg := rawExtra[i]
		if arg == "-m" || arg == "--model" {
			if i+1 < len(rawExtra) {
				model = rawExtra[i+1]
				i++
				continue
			}
		} else if strings.HasPrefix(arg, "--model=") {
			model = strings.TrimPrefix(arg, "--model=")
			continue
		} else if i == 0 && (arg == "chat" || arg == "v3" || arg == "r1" || arg == "reasoner" || strings.HasPrefix(arg, "deepseek-")) {
			model = arg
			continue
		}
		extra = append(extra, arg)
	}

	opts := launcher.Options{
		Port:      9655,
		Model:     model,
		ExtraArgs: extra,
	}

	switch target {
	case "claude", "claude-code":
		if err := launcher.LaunchClaude(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "dsh", "deepseek-harness":
		if err := launcher.LaunchDSH(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "codex":
		if err := launcher.LaunchCodexDirect(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "opencode":
		if err := launcher.LaunchOpenCode(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "goose":
		if err := launcher.LaunchGoose(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "aider":
		if err := launcher.LaunchAider(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "pool":
		if err := launcher.LaunchPool(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "chatgpt", "chatgpt-desktop":
		if err := launcher.LaunchChatGPT(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	case "harness", "bench", "test-harness":
		if err := launcher.LaunchHarness(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Bilinmeyen başlatıcı platformu: '%s'.\n", target)
		launcher.PrintLaunchMenu(os.Stdout)
	}
}
