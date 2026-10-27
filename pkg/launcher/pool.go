package launcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nexus-cli/pkg/cli"
	"nexus-cli/pkg/deepseek"
)

// LaunchPool displays and monitors the Nexus multi-account pool.
func LaunchPool(ctx context.Context, opts Options) error {
	pool := deepseek.NewAccountPool(opts.AccountsDir)
	accounts := pool.List()

	fmt.Printf("\n%s%s=== NEXUS ÇOKLU HESAP HAVUZU İZLEYİCİSİ ===%s\n\n", cli.BrightCyan, cli.Bold, cli.Reset)
	if len(accounts) == 0 {
		fmt.Println("Havuzda etkin hesap bulunamadı.")
		fmt.Printf("Hesap eklemek için: %snexus auth add <ad>%s\n\n", cli.BrightCyan, cli.Reset)
		return nil
	}

	activeName := pool.GetActiveAccountName()
	fmt.Printf("Toplam hesap: %d | Etkin birincil hesap: %s%s%s\n\n", len(accounts), cli.BrightGreen, activeName, cli.Reset)
	fmt.Printf("%-20s %-15s %-12s %-25s\n", "HESAP", "DURUM", "BEKLEME", "E-POSTA")
	fmt.Println(strings.Repeat("-", 75))

	for _, acc := range accounts {
		nameTag := acc.Name
		if acc.Name == activeName {
			nameTag = fmt.Sprintf("%s* %s%s", cli.BrightGreen, acc.Name, cli.Reset)
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

		email := "Bilinmiyor"
		if acc.Profile != nil && acc.Profile.Email != "" {
			email = acc.Profile.Email
		}

		fmt.Printf("%-20s %-15s %-12s %-25s\n", nameTag, status, cdStr, email)
	}

	fmt.Println("\nKomutlar:")
	fmt.Println("  • nexus auth add <ad>     - hesap ekle")
	fmt.Println("  • nexus auth test         - tüm hesapları doğrula")
	fmt.Println("  • nexus auth switch <ad>  - etkin hesabı değiştir")
	fmt.Println()

	// If interactive monitoring is requested
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(1 * time.Second):
		return nil
	}
}
