package repositories

import (
	"context"
	"errors"
	"sync"
	"time"

	cu "github.com/Davincible/chromedp-undetected"
	"github.com/chromedp/chromedp"
	"github.com/patrickmn/go-cache"
)

type VerifyRepository interface {
	VerifyByUrl(url string) (bool, error)
}

type verifyRepository struct {
}

var urlCache = cache.New(5*time.Minute, 10*time.Minute)

func NewVerifyRepository() VerifyRepository {
	return &verifyRepository{}
}

func (r *verifyRepository) VerifyByUrl(url string) (paymentSuccess bool, err error) {
	if _, found := urlCache.Get(url); found {
		return false, errors.New("URL is being processed")
	}
	ctx, cancel, err := cu.New(cu.NewConfig())
	if err != nil {
		return
	}
	defer cancel()
	urlCache.Set(url, true, cache.DefaultExpiration)
	defer urlCache.Delete(url)

	ctxT, cancelT := context.WithTimeout(ctx, 30*time.Second)
	defer cancelT()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if errT := chromedp.Run(ctxT,
			chromedp.WaitVisible(`*:contains("Payment Failed!")`, chromedp.ByQuery),
		); errT != nil {
			err = errT
			return
		}
	}()
	go func() {
		defer wg.Done()
		if errT := chromedp.Run(ctxT,
			chromedp.WaitVisible(`*:contains("Payment Complete!")`, chromedp.ByQuery)); errT != nil {
			err = errT
			return
		}
		paymentSuccess = true
	}()
	wg.Wait()
	if paymentSuccess {
		return true, nil
	}
	return
}
