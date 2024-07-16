package cards

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

type Card struct {
	Secret   string
	Key      string
	Currency string
}

type Transaction struct {
	TransactionStatusId int
	Amount              int
	Currency            string
	LastFour            string
	BankReturnCode      string
}

func (c *Card) Charge(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	return c.CreatePaymentIntent(currency, amount)
}

func (c *Card) CreatePaymentIntent(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	stripe.Key = c.Secret

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
	}

	pi, err := paymentintent.New(params)

	if err != nil {
		msg := ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = cardErrorMessage(stripeErr.Code)
		}
		return nil, msg, err
	}

	return pi, "", nil
}

func cardErrorMessage(code stripe.ErrorCode) string {
	var msg = ""

	switch code {
	case stripe.ErrorCodeCardDeclined:
		msg = "Card was declined"
	case stripe.ErrorCodeExpiredCard:
		msg = "Card is Expired"
	case stripe.ErrorCodeIncorrectCVC:
		msg = "Bad CVC"
	case stripe.ErrorCodeIncorrectZip:
		msg = "Bad zip code"
	case stripe.ErrorCodeAmountTooLarge:
		msg = "Amount was too large"
	case stripe.ErrorCodeAmountTooSmall:
		msg = "Amount was too small"
	case stripe.ErrorCodeBalanceInsufficient:
		msg = "Balance insufficient"
	case stripe.ErrorCodePostalCodeInvalid:
		msg = "Bad Postal Code"
	default:
		msg = "Error"
	}

	return msg
}
