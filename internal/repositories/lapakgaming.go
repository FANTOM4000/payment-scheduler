package repositories

import (
	"app/internal/domains"
	"app/internal/ports"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	cu "github.com/Davincible/chromedp-undetected"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	goqr "github.com/liyue201/goqr"
	"github.com/patrickmn/go-cache"
	"github.com/shopspring/decimal"
	_ "image/jpeg"
	_ "image/png"
)

type lapakgaming struct {
	id            string
	email         string
	ctx           context.Context
	cancelFunc    context.CancelFunc
	signalTapOpen <-chan target.ID
}



var acceptablePrice = map[int64]bool{
	50:   true,
	75:   true,
	100:  true,
	200:  true,
	350:  true,
	1000: true,
	2000: true,
}

var acceptablePaymentMethod = map[domains.PaymentMethod]string{
	domains.PromptPay: "PromptPay",
}

func NewLapakGaming(email string) ports.PaymentRepository {
	return &lapakgaming{
		email: email,
	}
}

func (l *lapakgaming) NewPayment(id string) (ports.PaymentRepository, error) {
	ctx, cancel, err := cu.New(cu.NewConfig(
		cu.WithChromeFlags(chromedp.Flag("disable-popup-blocking", true)),
	))
	if err != nil {
		log.Println("Failed to create Chrome instance:", err)
		return nil, err
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.lapakgaming.com/th-th/voucher-steam-wallet"),
	); err != nil {
		log.Println("Failed to navigate:", err)
		return nil, err
	}
	signalTapOpen := chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
		// เช็คว่าเป็น popup/แท็บใหม่
		return info.URL != "" && info.Type == "page"
	})
	ll := &lapakgaming{
		id:            id,
		email:         l.email,
		ctx:           ctx,
		cancelFunc:    cancel,
		signalTapOpen: signalTapOpen,
	}
	chromdpWorker.Set(id, ll, cache.DefaultExpiration)

	return ll, nil
}

func (l *lapakgaming) SubmitPayment(id string, method domains.PaymentMethod, phone string, amount decimal.Decimal, callBackProgress func(uint)) (urlRedirect, qrData, message, orderid string, err error) {
	if !acceptablePrice[amount.IntPart()] {
		err = errors.New("amount not acceptable")
		return
	}
	if err = chromedp.Run(l.ctx,
		chromedp.WaitReady(fmt.Sprintf(`//p[@data-testid="lgcardproduct-product-name" and normalize-space(text())="Steam Wallet Code THB %d"]`, amount.IntPart())),
		chromedp.Click(fmt.Sprintf(`//p[@data-testid="lgcardproduct-product-name" and normalize-space(text())="Steam Wallet Code THB %d"]`, amount.IntPart())),
	); err != nil {
		return 
	}

	paymentMethod, ok := acceptablePaymentMethod[method]
	if !ok {
		err = errors.New("payment method not acceptable")
		return
	}
	if err = chromedp.Run(l.ctx,
		chromedp.WaitReady(fmt.Sprintf(`//p[@class="text-xs ml-2 mt-1" and normalize-space(text())="%s"]`, paymentMethod)),
		chromedp.Click(fmt.Sprintf(`//p[@class="text-xs ml-2 mt-1" and normalize-space(text())="%s"]`, paymentMethod)),
	); err != nil {
		return
	}

	if err = chromedp.Run(l.ctx,
		chromedp.Clear(`input#phoneNumber`, chromedp.ByQuery),
		chromedp.SendKeys(`input#phoneNumber`, "999999999", chromedp.ByQuery),
		chromedp.Clear(`input#email`, chromedp.ByQuery),
		chromedp.SendKeys(`input#email`, l.email, chromedp.ByQuery),
		chromedp.WaitReady(`//button[@data-testid="lgpdpstickysummary-lgbuttonav-order" and normalize-space(text())="ซื้อเดี๋ยวนี้"]`),
		chromedp.Click(`//button[@data-testid="lgpdpstickysummary-lgbuttonav-order" and normalize-space(text())="ซื้อเดี๋ยวนี้"]`),
		chromedp.Sleep(2*time.Second), // รอให้ป๊อปอัพโหลด)
		chromedp.WaitReady(`//button[@data-testid="lgpdpconfirmationpopup-lgbuttonav" and normalize-space(text())="ชำระเดี๋ยวนี้"]`),
		chromedp.Click(`//button[@data-testid="lgpdpconfirmationpopup-lgbuttonav" and normalize-space(text())="ชำระเดี๋ยวนี้"]`),
		chromedp.Text(`//button[@data-testid="lgbuttoncopymv-text"]/preceding-sibling::div[1]`, &orderid, chromedp.BySearch),
	); err != nil {
		return
	}
	callBackProgress(10)
	orderid = strings.ReplaceAll(orderid, "#", "")
	select {
	case newTarget := <-l.signalTapOpen:
		// Attach ไปที่แท็บใหม่
		newCtx, cancelNew := chromedp.NewContext(l.ctx, chromedp.WithTargetID(newTarget))
		defer cancelNew()
		callBackProgress(20)
		// รอโหลด QR Code
		time.Sleep(2 * time.Second) // หรือใช้ chromedp.WaitVisible ถ้ามี selector ที่แน่นอน
		callBackProgress(30)
		var qrBase64 string
		err = chromedp.Run(newCtx,
			chromedp.WaitReady(`img[alt="QR image"]`), // รอให้ QR image ปรากฏ
			// ตัวอย่าง: ดึง text หรือ src ของ QR image
			chromedp.AttributeValue(`img[alt="QR image"]`, "src", &qrBase64, nil),
		)
		if err != nil {
			message = "Failed to retrieve QR code"
			return
		}
		callBackProgress(40)
		base64Str := strings.TrimPrefix(qrBase64, "data:image/png;base64,")
		base64Str = strings.TrimPrefix(base64Str, "data:image/jpeg;base64,")
		imgBytes, decodeErr := base64.StdEncoding.DecodeString(base64Str)
		if decodeErr != nil {
			err = decodeErr
			return
		}
		callBackProgress(45)
		img, _, decodeErr := image.Decode(bytes.NewReader(imgBytes))
		if decodeErr != nil {
			err = decodeErr
			return
		}
		callBackProgress(50)
		codes, scanErr := goqr.Recognize(img)
		if scanErr != nil {
			err = scanErr
			return
		}
		callBackProgress(55)
		if len(codes) == 0 {
			err = errors.New("no QR code found in image")
			return
		}
		callBackProgress(60)
		// use the first detected QR code payload
		qrData = string(codes[0].Payload)
		return
	case <-time.After(time.Minute):
		message = "Timeout waiting for tap open signal"
		err = context.DeadlineExceeded
	}
	return
}

func (l *lapakgaming) SubmitOtp(id string, otp string) (urlRedirect, qrData, message string, err error) {
	if err := chromedp.Run(l.ctx); err != nil {
		return "", "", "", err
	}
	return "", "", "", nil
}

func (l *lapakgaming) Close() {
	if l.cancelFunc != nil {
		l.cancelFunc()
	}
	chromdpWorker.Delete(l.id)
}
