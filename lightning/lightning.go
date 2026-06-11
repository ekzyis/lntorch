package lightning

import (
	"encoding/hex"

	"github.com/ekzyis/lnpilot/lightning/bolt11"
)

// Invoice is a bolt11 payment request plus its bech32 encoding and paid status.
// The encoding is carried alongside the request because a node returns it
// directly and it can't be reproduced from the request without the signing key.
type Invoice struct {
	*bolt11.PaymentRequest
	Bolt11 string
	Paid   bool
}

// Hash is the payment hash as hex, used for lookups, URLs and the database.
func (inv *Invoice) Hash() string {
	return hex.EncodeToString(inv.PaymentHash[:])
}

// Lightning mints invoices and reports whether they've been paid. Mock fakes
// payment; an LND-backed implementation will satisfy the same interface.
type Lightning interface {
	CreateInvoice(msats int64, description string) (*Invoice, error)
	GetInvoice(paymentHash string) (*Invoice, error)
}
