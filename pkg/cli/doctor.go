package cli

import (
	"context"
	"fmt"
	"time"

	"nexus-cli/pkg/deepseek"
)

// RunDoctor performs end-to-end diagnostics on DeepSeek API connectivity, auth, and PoW.
func RunDoctor(ctx context.Context, client *deepseek.Client) error {
	fmt.Printf("%s%s=== Nexus-CLI / DeepSeek Tanılama ===%s\n\n", BrightCyan, Bold, Reset)

	allOK := true

	// 1. Auth File & Token Check
	auth := client.Auth()
	token := auth.GetToken()
	data := auth.GetData()

	fmt.Print("[1/4] Kimlik doğrulama belirteci denetleniyor: ")
	if token == "" {
		fmt.Printf("%s❌ Belirteç bulunamadı!%s\n", Red, Reset)
		fmt.Println("      Çözüm:")
		fmt.Println("      1. 'nexus auth login' komutunu çalıştırın")
		fmt.Println("      2. Ya da .env dosyasına DEEPSEEK_TOKEN=belirteç_değeriniz yazın")
		allOK = false
	} else {
		masked := token
		if len(token) > 12 {
			masked = token[:6] + "..." + token[len(token)-4:]
		}
		fmt.Printf("%s✅ Mevcut (%s)%s\n", BrightGreen, masked, Reset)
		if data.Cookie != "" {
			fmt.Printf("      Çerezler: %s✅ Var (%d karakter)%s\n", Gray, len(data.Cookie), Reset)
		}
	}

	// 2. Validate Token via DeepSeek API
	fmt.Print("[2/4] DeepSeek hesap bilgileri doğrulanıyor: ")
	if token != "" {
		profile, err := auth.ValidateToken(ctx)
		if err != nil {
			fmt.Printf("%s❌ Hata (%v)%s\n", Red, err, Reset)
			allOK = false
		} else {
			name := profile.Nickname
			if name == "" {
				name = profile.Email
			}
			if name == "" {
				name = profile.ID
			}
			fmt.Printf("%s✅ Başarılı! Kullanıcı: %s (ID: %s)%s\n", BrightGreen, name, profile.ID, Reset)
		}
	} else {
		fmt.Printf("%s⚠️ Atlandı (belirteç yok)%s\n", Yellow, Reset)
	}

	// 3. PoW Challenge & Solver Speed
	fmt.Print("[3/4] PoW (iş kanıtı) sınaması ve Keccak hızı: ")
	if token != "" {
		start := time.Now()
		powResp, err := client.RequestAndSolvePoW(ctx, "/api/v0/chat/completion")
		duration := time.Since(start)
		if err != nil {
			fmt.Printf("%s❌ PoW çözülemedi (%v)%s\n", Red, err, Reset)
			allOK = false
		} else {
			fmt.Printf("%s✅ Başarılı! Çözüm süresi: %v (B64 uzunluğu: %d)%s\n", BrightGreen, duration, len(powResp), Reset)
		}
	} else {
		fmt.Printf("%s⚠️ Atlandı%s\n", Yellow, Reset)
	}

	// 4. Session Creation
	fmt.Print("[4/4] Sohbet oturumu oluşturuluyor (/api/v0/chat_session/create): ")
	if token != "" {
		sessionID, err := client.CreateSession(ctx)
		if err != nil {
			fmt.Printf("%s❌ Oturum oluşturulamadı (%v)%s\n", Red, err, Reset)
			allOK = false
		} else {
			fmt.Printf("%s✅ Oturum oluşturuldu (ID: %s)%s\n", BrightGreen, sessionID, Reset)
		}
	} else {
		fmt.Printf("%s⚠️ Atlandı%s\n", Yellow, Reset)
	}

	fmt.Println()
	if allOK {
		fmt.Printf("%s🎉 Tüm denetimler başarılı! Program kullanıma hazır.%s\n", BrightGreen, Reset)
	} else {
		fmt.Printf("%s⚠️ Bazı denetimlerde hata bulundu. Yukarıdaki yönergeleri inceleyin.%s\n", Yellow, Reset)
	}

	return nil
}
