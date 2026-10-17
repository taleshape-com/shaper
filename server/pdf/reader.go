// Credits to: https://gist.github.com/Seb-C/f458a99a3c7c913eed46df6ae471eac4
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"

	chromeio "github.com/chromedp/cdproto/io"
	"github.com/chromedp/chromedp"
)

var _ io.Reader = &ChromeReadAdapter{} // Static type check

// Adapter to stream a PDF from chrome protocol to a native Golang reader
// See https://github.com/chromedp/chromedp/issues/1489
type ChromeReadAdapter struct {
	ctx        context.Context
	handle     chromeio.StreamHandle
	batchSize  int
	buffer     *bytes.Buffer
	eofReached bool
}

// Initializes an adapter to stream PDFs from the chromedp
// The batch size does not reflect the exact memory usage,
// but is a rough estimate of it since the data has to be
// internally decoded from base64.
func NewChromeReadAdapter(
	ctx context.Context,
	handle chromeio.StreamHandle,
	batchSize int,
) *ChromeReadAdapter {
	return &ChromeReadAdapter{
		ctx:        ctx,
		handle:     handle,
		batchSize:  batchSize,
		buffer:     new(bytes.Buffer),
		eofReached: false,
	}
}

func (adapter *ChromeReadAdapter) Read(output []byte) (int, error) {
	if !adapter.eofReached && adapter.buffer.Len() < adapter.batchSize {
		res, err := chromedp.Call(adapter.ctx, chromeio.Read, chromeio.ReadParams{
			Handle: adapter.handle,
			Size:   int64(adapter.batchSize),
		})
		if err != nil {
			return 0, fmt.Errorf("read chrome stream: %w", err)
		}
		if res.EOF {
			adapter.eofReached = true
		}

		adapter.buffer.Write(res.Data)
	}

	return adapter.buffer.Read(output)
}
