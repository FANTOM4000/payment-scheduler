package domains

type PaymentMethod string

const (
	PromptPay       PaymentMethod = "promptpay"
	TrueMoneyWallet PaymentMethod = "truemoneywallet"

	TrueMoneyCode PaymentMethod = "truemoneycode"
	RazorGoldPin  PaymentMethod = "razorgoldpin"
)
