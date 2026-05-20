package lightning

import (
	"time"

	"github.com/ekzyis/lnpilot/lightning/bolt11"
)

type Lightning interface {
	CreateInvoice(msats int64, description string, expiresAt time.Time) (*bolt11.PaymentRequest, error)
	GetInvoice(paymentHash string) (*bolt11.PaymentRequest, error)
}
