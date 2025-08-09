package bot

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func ScrapeFacebookGroup(pageURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/brave"),
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	cookies, err := loadCookies("facebook-cookies.txt")
	if err != nil {
		return "", fmt.Errorf("gagal memuat cookies: %w. Pastikan facebook-cookies.txt ada", err)
	}

	var imageURL string
	tasks := chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookieParams := []*network.CookieParam{}
			for _, cookie := range cookies {
				cookieParams = append(cookieParams, &network.CookieParam{
					Name:   cookie.Name,
					Value:  cookie.Value,
					Domain: cookie.Domain,
				})
			}
			return network.SetCookies(cookieParams).Do(ctx)
		}),
		chromedp.Navigate(pageURL),

		// --- LOGIKA BARU UNTUK MENGATASI HALAMAN YANG "DIAM" ---
		// 1. Beri waktu sejenak agar halaman memuat elemen awal
		chromedp.Sleep(5 * time.Second),

		// 2. Coba cari dan klik tombol "close" pada pop-up login/cookie yang sering muncul.
		// '[aria-label="Close"]' adalah selector umum untuk tombol X.
		// Kita gunakan 'chromedp.Run' di sini agar tidak error jika tombolnya tidak ada.
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Mencoba menutup pop-up yang mungkin muncul...")
			_ = chromedp.Run(ctx, chromedp.Click(`[aria-label="Close"]`, chromedp.NodeVisible, chromedp.ByQuery))
			return nil
		}),

		// 3. Tunggu hingga elemen gambar utama benar-benar terlihat
		chromedp.WaitVisible(`img[data-visualcompletion="media-vc-image"]`, chromedp.ByQuery),

		// 4. Baru ambil URL gambarnya
		chromedp.AttributeValue(`img[data-visualcompletion="media-vc-image"]`, "src", &imageURL, nil, chromedp.ByQuery),
		// -----------------------------------------------------
	}

	if err := chromedp.Run(taskCtx, tasks); err != nil {
		return "", fmt.Errorf("gagal menjalankan browser otomatis: %w", err)
	}

	if imageURL == "" {
		return "", fmt.Errorf("tidak dapat menemukan URL gambar di halaman, mungkin struktur halaman berubah atau konten tidak tersedia")
	}

	return downloadDirectImage(imageURL)
}

// ... (Sisa file loadCookies dan downloadDirectImage tetap sama)
func loadCookies(filename string) ([]*http.Cookie, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var cookies []*http.Cookie

	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) == 7 {
			cookies = append(cookies, &http.Cookie{
				Domain: parts[0],
				Name:   parts[5],
				Value:  parts[6],
			})
		}
	}
	return cookies, nil
}

func downloadDirectImage(url string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "aether-scrape-")
	if err != nil {
		return "", err
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d.jpg", time.Now().UnixNano()))
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
