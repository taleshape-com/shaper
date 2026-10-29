// SPDX-License-Identifier: MPL-2.0

package pdf

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	chromeio "github.com/chromedp/cdproto/io"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/golang-jwt/jwt/v5"
)

var (
	PDF_STREAM_BATCH_SIZE = 1024 * 32 // 32kb
	PDF_SCALE             = 0.8341
	BROWSER_WIDTH         = 1020 // Matches print body width in index.css so charts are at correct size for print
	BROWSER_HEIGHT        = 500  // Limit height so cards use min-height
	PNG_BROWSER_WIDTH     = 670
)

func StreamDashboardPdf(
	ctx context.Context,
	logger *slog.Logger,
	writer io.Writer,
	baseUrl string,
	pdfDateFormat string,
	dashboardId string,
	params url.Values,
	jwtToken *jwt.Token,
	noChromeSandbox bool,
) error {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(int(BROWSER_WIDTH), int(BROWSER_HEIGHT)),
	)
	if noChromeSandbox {
		opts = append(opts, chromedp.NoSandbox)
	}
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// capture pdf
	vars, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to json marshal params: %w", err)
	}
	urlstr := baseUrl + "/_internal/pdfview/" + url.PathEscape(dashboardId) + "?vars=" + base64.StdEncoding.EncodeToString(vars) + "#jwt=" + jwtToken.Raw
	headerImage := ""
	footerLink := ""
	err = chromedp.Do(ctx,
		chromedp.Navigate(urlstr),
		// Wait for either the dashboard to load or an error to appear
		chromedp.WaitReady(`.shaper-scope .shaper-custom-dashboard-header, #shaper-error-message`),
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			// Check if error message is present
			nodes, err := chromedp.Nodes("#shaper-error-message", chromedp.AtLeast(0))(ctx, t)
			if err != nil {
				return err
			}
			if len(nodes) > 0 {
				errorMessage, err := chromedp.Text("#shaper-error-message")(ctx, t)
				if err != nil {
					return err
				}
				return fmt.Errorf("UI Error: %s", errorMessage)
			}
			return nil
		}),
		// Get header image and footer link from dashboard
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			headerRes, err := chromedp.AttributeValue(`.shaper-scope .shaper-custom-dashboard-header`, "data-header-image")(ctx, t)
			if err != nil {
				return err
			}
			headerImage = headerRes.Value

			footerRes, err := chromedp.AttributeValue(`.shaper-scope .shaper-custom-dashboard-footer`, "data-footer-link")(ctx, t)
			if err != nil {
				return err
			}
			footerLink = footerRes.Value
			return nil
		}),
		// Set margin-top if header image is set to avoid overlap
		// Also remove header and footer elements as we use PDF header+footer
		chromedp.Evaluate[chromedp.Void](`
			const el = document.querySelector('.shaper-scope .shaper-custom-dashboard-header');
			if (el && el.getAttribute('data-header-image')) {
				const style = document.createElement('style');
				style.textContent = '@page { margin-top: 20mm; }';
				document.head.appendChild(style);
			}
			const header = document.querySelector('.shaper-scope .shaper-custom-dashboard-header');
			if (header) header.remove();
			const footer = document.querySelector('.shaper-scope .shaper-custom-dashboard-footer');
			if (footer) footer.remove();
		`),
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			h, err := header(headerImage)
			if err != nil {
				logger.Warn("failed to load header image for pdf", slog.String("image", headerImage), slog.String("dashboard", dashboardId), slog.Any("error", err))
			}
			res, err := cdp.Call(ctx, t, page.PrintToPDF, page.PrintToPDFParams{
				PrintBackground:     ptr(true),
				PreferCSSPageSize:   ptr(true),
				Scale:               PDF_SCALE,
				DisplayHeaderFooter: ptr(true),
				HeaderTemplate:      h,
				FooterTemplate:      footer(time.Now().Format(pdfDateFormat), footerLink),
				TransferMode:        page.PrintToPDFTransferModeReturnAsStream,
			})
			if err != nil {
				return err
			}
			rawPdfStream := res.Stream
			defer func() { _, _ = cdp.Call(ctx, t, chromeio.Close, chromeio.CloseParams{Handle: rawPdfStream}) }()

			pdfReader := NewChromeReadAdapter(
				ctx,
				rawPdfStream,
				PDF_STREAM_BATCH_SIZE,
			)
			_, err = io.Copy(writer, pdfReader)
			return err
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to generate pdf: %w", err)
	}
	return nil
}

func StreamDashboardPng(
	ctx context.Context,
	logger *slog.Logger,
	writer io.Writer,
	baseUrl string,
	dashboardId string,
	params url.Values,
	jwtToken *jwt.Token,
	noChromeSandbox bool,
) error {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(int(PNG_BROWSER_WIDTH), int(BROWSER_HEIGHT)),
	)
	if noChromeSandbox {
		opts = append(opts, chromedp.NoSandbox)
	}
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// capture png
	vars, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to json marshal params: %w", err)
	}
	urlstr := baseUrl + "/_internal/pdfview/" + url.PathEscape(dashboardId) + "?vars=" + base64.StdEncoding.EncodeToString(vars) + "#jwt=" + jwtToken.Raw
	pngData := ""
	err = chromedp.Do(ctx,
		chromedp.Navigate(urlstr),
		// Wait for either the dashboard to load or an error to appear
		chromedp.WaitReady(`.shaper-scope .shaper-custom-dashboard-header, #shaper-error-message`),
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			// Check if error message is present
			nodes, err := chromedp.Nodes("#shaper-error-message", chromedp.AtLeast(0))(ctx, t)
			if err != nil {
				return err
			}
			if len(nodes) > 0 {
				errorMessage, err := chromedp.Text("#shaper-error-message")(ctx, t)
				if err != nil {
					return err
				}
				return fmt.Errorf("UI Error: %s", errorMessage)
			}
			return nil
		}),
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			// Assert that exactly 1 download button is present
			nodes, err := chromedp.Nodes(".shaper-chart-download-button", chromedp.AtLeast(0))(ctx, t)
			if err != nil {
				return err
			}
			if len(nodes) == 0 {
				return fmt.Errorf("PNG download is only supported for dashboards with 1 chart (found 0)")
			}
			if len(nodes) > 1 {
				return fmt.Errorf("PNG download is only supported for dashboards with exactly 1 chart (found %d)", len(nodes))
			}
			return nil
		}),
		// Monkey-patch download behavior
		chromedp.Evaluate[chromedp.Void](`
			window.shaper_captured_png = null;
			const originalClick = HTMLAnchorElement.prototype.click;
			HTMLAnchorElement.prototype.click = function() {
				if (this.download && this.href.startsWith('data:image/png')) {
					window.shaper_captured_png = this.href;
					return;
				}
				originalClick.apply(this, arguments);
			};
		`),
		// Trigger the click
		chromedp.Click(".shaper-chart-download-button"),
		// Poll for the captured png
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			timeout := time.After(10 * time.Second)
			for {
				select {
				case <-ticker.C:
					captured, err := chromedp.Evaluate[string](`window.shaper_captured_png || ""`)(ctx, t)
					if err != nil {
						return err
					}
					if captured != "" {
						pngData = captured
						return nil
					}
				case <-timeout:
					return fmt.Errorf("timeout waiting for PNG download")
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to generate png: %w", err)
	}

	// pngData is a data URL like data:image/png;base64,...
	parts := strings.Split(pngData, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid png data url")
	}
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to decode png base64: %w", err)
	}

	_, err = writer.Write(data)
	return err
}

func header(imageURL string) (string, error) {
	if imageURL == "" {
		return "<span></span>", nil
	}

	// Download and convert image to base64 data URL
	base64Image, err := downloadImageAsBase64(imageURL)
	if err != nil {
		return "<span></span>", err
	}

	return `<img src="` + base64Image + `" style="max-height: 40px; margin-left: 35px; object-fit: contain;" />`, nil
}

func downloadImageAsBase64(imageURL string) (string, error) {
	if strings.HasPrefix(imageURL, "data:") {
		return imageURL, nil
	}

	parsed, err := url.Parse(imageURL)
	if err != nil {
		return "", fmt.Errorf("invalid image URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported image URL scheme: %s", parsed.Scheme)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	const maxImageSize = 10 << 20 // 10 MB
	imageData, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize))
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Determine content type from URL extension or response header
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback to guessing from file extension
		ext := strings.ToLower(filepath.Ext(imageURL))
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		default:
			contentType = "image/jpeg" // Default fallback
		}
	}

	// Convert to base64
	base64String := base64.StdEncoding.EncodeToString(imageData)

	// Return as data URL
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64String), nil
}

func footer(formattedDate string, footerLink string) string {
	content := ""
	if footerLink != "" {
		formattedLink := strings.TrimPrefix(footerLink, "http://")
		formattedLink = strings.TrimPrefix(formattedLink, "https://")
		formattedLink = strings.TrimPrefix(formattedLink, "mailto:")
		content = `<a href="` + html.EscapeString(footerLink) + `" style="color: var(--shaper-text-color-secondary); text-decoration: none;">` + html.EscapeString(formattedLink) + `</a>`
	}
	return `
<div style="font-family: var(--shaper-font), ui-sans-serif, system-ui, sans-serif; font-size:7px; width: 100%; margin: 0 35px; display: flex; justify-content: space-between;">
	<span style="color: var(--shaper-text-color-secondary);">` + formattedDate + `</span>
	` + content + `
	<span style="color: var(--shaper-text-color-secondary);">
		<span class="pageNumber"></span>
		<span>/</span>
		<span class="totalPages"></span>
	</span>
</div>`
}

func ptr[T any](v T) *T {
	return &v
}
