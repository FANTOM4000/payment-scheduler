package repositories

import (
	"app/internal/domains"
	"app/internal/ports"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"log"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	cu "github.com/Davincible/chromedp-undetected"
	"github.com/chromedp/chromedp"
	goqr "github.com/liyue201/goqr"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"github.com/shopspring/decimal"
)

type seagm struct {
	mainCtx        context.Context
	mainCancelFunc context.CancelFunc

	tabCtx        context.Context
	tabCancelFunc context.CancelFunc
}

func NewSeagm(email, password string) ports.PaymentRepository {
	ctx, cancel, err := cu.New(cu.NewConfig())
	if err != nil {
		return nil
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://member.seagm.com/en-th/sso/login"),
	); err != nil {
		return nil
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://member.seagm.com/en-th/sso/login"),
		chromedp.WaitVisible(`button[id="CybotCookiebotDialogBodyLevelButtonLevelOptinAllowAll"]`, chromedp.ByQuery),
		chromedp.Click(`button[id="CybotCookiebotDialogBodyLevelButtonLevelOptinAllowAll"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`input[id="login_email"]`, chromedp.ByQuery),
		chromedp.SetValue(`input[id="login_email"]`, email, chromedp.ByQuery),
		chromedp.SetValue(`input[id="login_pass"]`, password, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Just to see the result
		chromedp.WaitReady(`label[id="login_btw"]`, chromedp.ByQuery),
		chromedp.Click(`label[id="login_btw"]`, chromedp.ByQuery),
		chromedp.WaitReady(`div[id="main_nav"]`, chromedp.ByQuery),
		chromedp.Navigate("https://www.seagm.com/en-th/language_currency"),
		chromedp.WaitReady(`div.region_item[region="th"][region-currency="THB"]`, chromedp.ByQuery),
		chromedp.Click(`div.region_item[region="th"][region-currency="THB"]`, chromedp.ByQuery),
	); err != nil {
		return nil
	}

	browserCtx, cancelBrowser := chromedp.NewContext(ctx)

	cronjob := cron.New()
	cronjob.AddFunc("@hourly", func() {
		var currentURL string
		chromedp.Run(browserCtx,
			chromedp.Reload(),
			chromedp.Location(&currentURL),
		)
		if strings.Contains(currentURL, "https://www.seagm.com/en-th/sso/login") {
			if err := chromedp.Run(ctx,
				chromedp.Navigate("https://member.seagm.com/en-th/sso/login"),
				chromedp.WaitVisible(`button[id="CybotCookiebotDialogBodyLevelButtonLevelOptinAllowAll"]`, chromedp.ByQuery),
				chromedp.Click(`button[id="CybotCookiebotDialogBodyLevelButtonLevelOptinAllowAll"]`, chromedp.ByQuery),
				chromedp.WaitVisible(`input[id="login_email"]`, chromedp.ByQuery),
				chromedp.SetValue(`input[id="login_email"]`, email, chromedp.ByQuery),
				chromedp.SetValue(`input[id="login_pass"]`, password, chromedp.ByQuery),
				chromedp.Sleep(2*time.Second), // Just to see the result
				chromedp.WaitReady(`label[id="login_btw"]`, chromedp.ByQuery),
				chromedp.Click(`label[id="login_btw"]`, chromedp.ByQuery),
				chromedp.WaitReady(`div[id="main_nav"]`, chromedp.ByQuery),
				chromedp.Navigate("https://www.seagm.com/en-th/language_currency"),
				chromedp.WaitReady(`div.region_item[region="th"][region-currency="THB"]`, chromedp.ByQuery),
				chromedp.Click(`div.region_item[region="th"][region-currency="THB"]`, chromedp.ByQuery),
			); err != nil {
				log.Println("Error re-authenticating SEAGM:", err)
			}
		}
	})
	cronjob.Start()

	sg := &seagm{
		mainCtx: browserCtx,
		mainCancelFunc: func() {
			cancel()
			cancelBrowser()
			cronjob.Stop()
		},
	}

	return sg
}

func (sg *seagm) NewPayment(id string) (ports.PaymentRepository, error) {
	tabCtx, cancelTab := chromedp.NewContext(sg.mainCtx)
	sgg := &seagm{
		mainCtx:        sg.mainCtx,
		mainCancelFunc: sg.mainCancelFunc,
		tabCtx:         tabCtx,
		tabCancelFunc:  cancelTab,
	}
	chromdpWorker.Set(id, sgg, cache.DefaultExpiration)
	return sgg, nil
}

func (sg *seagm) SubmitPayment(id string, method domains.PaymentMethod, phone string, amount decimal.Decimal, callBackProgress func(uint)) (urlRedirect, qrData, message, orderid string, err error) {

	//set amount

	if err = chromedp.Run(sg.tabCtx,
		chromedp.Navigate("https://www.seagm.com/en-th/ucp/topup"),
	); err != nil {
		return
	}

	if err = chromedp.Run(sg.tabCtx,
		chromedp.WaitReady(`input#top_up_amount`, chromedp.ByQuery),
		chromedp.SetValue(`input#top_up_amount`, amount.String(), chromedp.ByQuery),
		chromedp.Click(`input#submit`, chromedp.ByQuery),
	); err != nil {
		return
	}

	if err = chromedp.Run(sg.tabCtx,
		chromedp.Sleep(2*time.Second), // Just to see the result
		chromedp.WaitReady(`div.channel[data-method-code="promptpay_qr"]`, chromedp.ByQuery),
		chromedp.Click(`div.channel[data-method-code="promptpay_qr"]`, chromedp.ByQuery),
	); err != nil {
		return
	}

	if err = chromedp.Run(sg.tabCtx,
		chromedp.WaitReady(`label.paynow.btw`, chromedp.ByQuery),
		chromedp.Click(`label.paynow.btw`, chromedp.ByQuery),
	); err != nil {
		return
	}

	var qrBase64 string
	err = chromedp.Run(sg.tabCtx,
		chromedp.WaitReady(`img[alt="QR image"]`),
		chromedp.AttributeValue(`img[alt="QR image"]`, "src", &qrBase64, nil),
	)
	if err != nil {
		return
	}
	base64Str := strings.TrimPrefix(qrBase64, "data:image/png;base64,")
	base64Str = strings.TrimPrefix(base64Str, "data:image/jpeg;base64,")
	imgBytes, decodeErr := base64.StdEncoding.DecodeString(base64Str)
	if decodeErr != nil {
		return
	}
	img, _, decodeErr := image.Decode(bytes.NewReader(imgBytes))
	if decodeErr != nil {
		return
	}
	codes, scanErr := goqr.Recognize(img)
	if scanErr != nil {
		return
	}
	if len(codes) == 0 {
		err = errors.New("no QR code found in image")
		return
	}

	var currentURL string
	err = chromedp.Run(sg.tabCtx,
		chromedp.Location(&currentURL),
	)
	if err != nil {
		return
	}

	// use the first detected QR code payload
	qrData = string(codes[0].Payload)
	urlRedirect = currentURL
	return
}

func (sg *seagm) SubmitOtp(id string, otp string) (urlRedirect, qrData, message string, err error) {
	return "", "", "", nil
}

func (sg *seagm) Close() {
	if sg.tabCancelFunc != nil {
		sg.tabCancelFunc()
	} else {
		sg.mainCancelFunc()
	}
}
