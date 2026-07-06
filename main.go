package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const renderDelay = 3 * time.Second
const defaultPNGWidth = 183
const defaultPNGHeight = 1024
const pngUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

func chromeExecPath() string {
	if path := os.Getenv("CHROME_PATH"); path != "" {
		return path
	}

	for _, name := range []string{"chromium", "google-chrome", "chromium-browser"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path
		}
	}

	if _, err := os.Stat("/headless-shell/headless-shell"); err == nil {
		return "/headless-shell/headless-shell"
	}

	return "/usr/bin/chromium"
}

func renderPDF(body []byte) ([]byte, error) {
	userDataDir, err := os.MkdirTemp("", "chromedp-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(userDataDir)

	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromeExecPath()),
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-zygote", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-crash-reporter", true),
		chromedp.Flag("disable-crashpad", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("disable-features", "UseDBus"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserDataDir(userDataDir),
	)

	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), allocatorOptions...)
	defer cancelAllocator()

	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	var pdfBytes []byte
	err = chromedp.Run(ctx,
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}

			return page.SetDocumentContent(frameTree.Frame.ID, string(body)).Do(ctx)
		}),
		chromedp.Sleep(renderDelay),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBytes, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0).
				WithMarginRight(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}

	return pdfBytes, nil
}

func renderPNG(body []byte, widthPage, heightPage int64) ([]byte, error) {
	userDataDir, err := os.MkdirTemp("", "chromedp-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(userDataDir)

	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromeExecPath()),
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-zygote", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-crash-reporter", true),
		chromedp.Flag("disable-crashpad", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("disable-features", "UseDBus"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserDataDir(userDataDir),
	)

	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), allocatorOptions...)
	defer cancelAllocator()

	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	var pngBytes []byte
	err = chromedp.Run(ctx,
		emulation.SetUserAgentOverride(pngUserAgent),
		emulation.SetDeviceMetricsOverride(widthPage, heightPage, 1, false),
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}

			return page.SetDocumentContent(frameTree.Frame.ID, string(body)).Do(ctx)
		}),
		chromedp.FullScreenshot(&pngBytes, 100),
	)
	if err != nil {
		return nil, err
	}

	return pngBytes, nil
}

func convertHTMLToPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid or empty HTML", http.StatusBadRequest)
		return
	}

	token := os.Getenv("TOKEN")

	authorization := r.Header.Get("Authorization")

	if authorization != "Bearer "+token {
		http.Error(w, "Invalid authorization", http.StatusUnauthorized)
		return
	}

	pdfBytes, err := renderPDF(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate PDF: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=output.pdf")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}

func queryInt64(r *http.Request, name string, defaultValue int64) (int64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}

	return parsed, nil
}

func convertHTMLToPNG(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid or empty HTML", http.StatusBadRequest)
		return
	}

	token := os.Getenv("TOKEN")

	authorization := r.Header.Get("Authorization")

	if authorization != "Bearer "+token {
		http.Error(w, "Invalid authorization", http.StatusUnauthorized)
		return
	}

	widthPage, err := queryInt64(r, "width", defaultPNGWidth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	heightPage, err := queryInt64(r, "height", defaultPNGHeight)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pngBytes, err := renderPNG(body, widthPage, heightPage)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate PNG: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(base64.StdEncoding.EncodeToString(pngBytes)))
}

func serverPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}

	return "8080"
}

func main() {
	http.HandleFunc("/generate-pdf", convertHTMLToPDF)
	http.HandleFunc("/generate-png", convertHTMLToPNG)

	port := serverPort()
	fmt.Printf("Servidor iniciado em http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
