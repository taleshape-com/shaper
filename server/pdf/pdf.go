// SPDX-License-Identifier: MPL-2.0

package pdf

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	chromeio "github.com/chromedp/cdproto/io"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var PDF_STREAM_BATCH_SIZE = 1024 * 32 // 32kb

func StreamDashboardPdf(
	ctx context.Context,
	dashboardId string,
	params url.Values,
	variables map[string]any,
	writer io.Writer,
) error {
	scale := 0.75
	w := 1320/scale - 470
	h := 500 // Not too high size we want to have cards at min-height

	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(int(w), int(h)),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// capture pdf
	urlstr := `http://localhost:5454/view/` + dashboardId + "?" + params.Encode()
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.WaitVisible(`.shaper-scope section`),
		// chromedp.Sleep(500 * time.Millisecond), // TODO: waiting for fonts to load
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("printing to pdf...")
			_, rawPdfStream, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithScale(scale).
				WithDisplayHeaderFooter(true).
				WithHeaderTemplate(`<span></span>`).
				WithFooterTemplate(footer(time.Now())).
				WithTransferMode(page.PrintToPDFTransferModeReturnAsStream).
				Do(ctx)
			if err != nil {
				return err
			}
			defer chromeio.Close(rawPdfStream).Do(ctx)

			pdfReader := NewChromeReadAdapter(
				ctx,
				rawPdfStream,
				PDF_STREAM_BATCH_SIZE,
			)
			_, err = io.Copy(writer, pdfReader)
			return err
		}),
	})
	fmt.Println("wrote pdf")
	if err == nil {
		return fmt.Errorf("failed to generate pdf: %w", err)
	}
	return nil
}

func footer(date time.Time) string {
	return `
<div style="font-family: var(--shaper-font), ui-sans-serif, system-ui, sans-serif; font-size:8px; width: 100%; margin: 0 35px; display: flex; justify-content: space-between;">
	<span style="color: var(--shaper-text-color-secondary);">` + date.Format("02.01.2006") + `</span>
	<a style="color: var(--shaper-text-color); text-decoration: none;" href="url">url</a>
	<span style="color: var(--shaper-text-color-secondary);">
		<span class="pageNumber"></span>
		<span>/</span>
		<span class="totalPages"></span>
	</span>
</div>`
}
