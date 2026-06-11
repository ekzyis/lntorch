package lightning

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/ekzyis/lnpilot/lib/secp256k1"
	"github.com/ekzyis/lnpilot/lightning/bolt11"
)

// Mock is an in-memory Lightning that mints real (but unpayable) bolt11
// invoices signed with a throwaway key. With autopay on, a background loop
// settles every outstanding invoice every 5 seconds, so the payment flow can be
// exercised without a node.
type Mock struct {
	mu       sync.Mutex
	invoices map[string]*Invoice
}

func NewMock(autopay bool) *Mock {
	m := &Mock{invoices: make(map[string]*Invoice)}
	if autopay {
		go m.autopayLoop()
	}
	return m
}

func (m *Mock) autopayLoop() {
	for range time.Tick(5 * time.Second) {
		m.mu.Lock()
		for _, inv := range m.invoices {
			inv.Paid = true
		}
		m.mu.Unlock()
	}
}

func (m *Mock) CreateInvoice(msats int64, description string) (*Invoice, error) {
	key := make([]byte, 32)
	rand.Read(key)
	signer, err := secp256k1.NewPrivateKeySigner(key)
	if err != nil {
		return nil, err
	}

	pr, err := bolt11.NewPaymentRequest(uint64(msats),
		bolt11.WithDescription(description),
		bolt11.WithExpiry(time.Hour),
	)
	if err != nil {
		return nil, err
	}

	encoded, err := pr.EncodeBech32(signer)
	if err != nil {
		return nil, err
	}

	inv := &Invoice{PaymentRequest: pr, Bolt11: encoded}

	m.mu.Lock()
	m.invoices[inv.Hash()] = inv
	m.mu.Unlock()

	return inv, nil
}

// Pay settles a single invoice immediately; used by tests.
func (m *Mock) Pay(paymentHash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inv, ok := m.invoices[paymentHash]; ok {
		inv.Paid = true
	}
}

// PayAll settles every outstanding invoice and returns how many it settled.
func (m *Mock) PayAll() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, inv := range m.invoices {
		if !inv.Paid {
			inv.Paid = true
			n++
		}
	}
	return n
}

func (m *Mock) GetInvoice(paymentHash string) (*Invoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.invoices[paymentHash]
	if !ok {
		return nil, fmt.Errorf("invoice %s not found", paymentHash)
	}
	return inv, nil
}
