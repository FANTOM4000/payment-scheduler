package repositories

import (
	"app/internal/domains"
	"app/internal/ports"
	"context"
	"errors"
	"strings"
	"time"

	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"

	cu "github.com/Davincible/chromedp-undetected"
	"github.com/chromedp/chromedp"
	goqr "github.com/liyue201/goqr"
	"github.com/patrickmn/go-cache"
	"github.com/shopspring/decimal"
)

type ggkeystore struct {
	mainCtx        context.Context
	mainCancelFunc context.CancelFunc

	tabCtx        context.Context
	tabCancelFunc context.CancelFunc
}

func NewGgkeystore(email, password string) ports.PaymentRepository {
	ctx, cancel, err := cu.New(cu.NewConfig())
	if err != nil {
		return nil
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.ggkeystore.com/login"),
	); err != nil {
		return nil
	}

	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SetValue(`input[name="email"]`, email),
		chromedp.SetValue(`input[name="password"]`, password),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.WaitReady(`form[action="https://www.ggkeystore.com/logout"]`, chromedp.ByQuery),
	); err != nil {
		return nil
	}

	browserCtx, cancelBrowser := chromedp.NewContext(ctx)

	gg := &ggkeystore{
		mainCtx: browserCtx,
		mainCancelFunc: func() {
			cancel()
			cancelBrowser()
		},
	}

	return gg
}

func (g *ggkeystore) NewPayment(id string) (ports.PaymentRepository, error) {
	tabCtx, cancelTab := chromedp.NewContext(g.mainCtx)
	gg := &ggkeystore{
		mainCtx:        g.mainCtx,
		mainCancelFunc: g.mainCancelFunc,
		tabCtx:         tabCtx,
		tabCancelFunc:  cancelTab,
	}
	chromdpWorker.Set(id, gg, cache.DefaultExpiration)
	return gg, nil
}

func (g *ggkeystore) SubmitPayment(id string, method domains.PaymentMethod, phone string, amount decimal.Decimal, callBackProgress func(uint)) (urlRedirect, qrData, message, orderid string, err error) {
	err = chromedp.Run(g.tabCtx,
		chromedp.Navigate("https://www.ggkeystore.com/topup"),
	)
	if err != nil {
		return
	}

	err = chromedp.Run(g.tabCtx,
		chromedp.WaitVisible(`input#amount`, chromedp.ByQuery),
	)
	if err != nil {
		return
	}

	err = chromedp.Run(g.tabCtx,
		chromedp.WaitVisible(`input#amount`, chromedp.ByQuery),
		chromedp.SetValue(`input#amount`, amount.String(), chromedp.ByQuery),
	)
	if err != nil {
		return
	}

	err = chromedp.Run(g.tabCtx,
		chromedp.WaitVisible(`button[type="submit"].btn-success`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // wait for 1 second before clicking
		chromedp.Click(`button[type="submit"].btn-success`, chromedp.ByQuery),
	)
	if err != nil {
		return
	}

	foundImage := false
	qrBase64 := ""

	err = chromedp.Run(g.tabCtx,
		chromedp.WaitVisible(`//p[@class="channel" and contains(text(),"ชำระผ่านคิวอาร์")]`, chromedp.BySearch),
		chromedp.Click(`//p[@class="channel" and contains(text(),"ชำระผ่านคิวอาร์")]`, chromedp.BySearch),
		chromedp.WaitVisible(`//p[@class="channel" and contains(text(),"พร้อมเพย์")]`, chromedp.BySearch),
		chromedp.Click(`//p[@class="channel" and contains(text(),"พร้อมเพย์")]`, chromedp.BySearch),
		chromedp.WaitVisible(`img#qr-pay`, chromedp.ByQuery),
	)
	if err != nil {
		return
	}
	callBackProgress(20)
	err = chromedp.Run(g.tabCtx,
		chromedp.Sleep(2*time.Second), // wait for the QR code to load
		chromedp.WaitReady(`h1.box-merchant-payment-bar-info-h1`, chromedp.ByQuery),
		chromedp.Text(`h1.box-merchant-payment-bar-info-h1`, &orderid, chromedp.ByQuery),
	)
	if err != nil {
		return
	}
	callBackProgress(30)
	err = chromedp.Run(g.tabCtx,
		chromedp.WaitReady(`img#qr-pay`, chromedp.ByQuery),
		chromedp.WaitVisible(`img#qr-pay`, chromedp.ByQuery),
		chromedp.AttributeValue(`img#qr-pay`, "src", &qrBase64, &foundImage, chromedp.ByQuery),
	)
	if err != nil {
		return
	}
	callBackProgress(40)
	if !foundImage {
		err = errors.New("failed to find QR code image")
		return
	}
	// extract QR data from qrBase64 data:image/png;base64,iVBORw0K...
	// NOTE: requires adding imports:

	base64Str := strings.TrimPrefix(qrBase64, "data:image/png;base64,")
	base64Str = strings.TrimPrefix(base64Str, "data:image/jpeg;base64,")
	imgBytes, decodeErr := base64.StdEncoding.DecodeString(base64Str)
	if decodeErr != nil {
		err = decodeErr
		return
	}
	callBackProgress(50)
	img, _, decodeErr := image.Decode(bytes.NewReader(imgBytes))
	if decodeErr != nil {
		err = decodeErr
		return
	}
	codes, scanErr := goqr.Recognize(img)
	if scanErr != nil {
		err = scanErr
		return
	}
	callBackProgress(60)
	if len(codes) == 0 {
		err = errors.New("no QR code found in image")
		return
	}
	// use the first detected QR code payload
	qrData = string(codes[0].Payload)
	callBackProgress(70)
	return
}

func (g *ggkeystore) SubmitOtp(id string, otp string) (urlRedirect, qrData, message string, err error) {
	err = chromedp.Run(g.tabCtx,
		chromedp.WaitVisible(`input#otp`, chromedp.ByQuery),
		chromedp.SetValue(`input#otp`, otp, chromedp.ByQuery),
		chromedp.Click(`button[type="submit"].btn-success`, chromedp.ByQuery),
	)
	return
}

func (g *ggkeystore) Close() {
	if g.tabCancelFunc != nil {
		g.tabCancelFunc()
	} else {
		g.mainCancelFunc()
	}
}
