package token

import (
	"errors"
	"fmt"
	"log"
	"time"
)

func (t *Token) Mint(to string, amount uint64) error {
	log.Printf("Minting %d tokens to %s", amount, to)

	// Check if operations are paused (unless emergency mode)
	if !t.IsEmergencyMode() {
		if err := t.requireNotPaused(); err != nil {
			log.Printf("Mint failed: %v", err)
			return err
		}
	}

	// Input validation
	if !t.validateAddress(to) {
		err := errors.New("invalid address")
		log.Printf("Mint failed: %v", err)
		LogTokenOperation(t, "mint", "", to, amount, "failed", map[string]interface{}{"error": err.Error()})
		return err
	}
	if amount == 0 {
		err := errors.New("amount must be > 0")
		log.Printf("Mint failed: %v", err)
		LogTokenOperation(t, "mint", "", to, amount, "failed", map[string]interface{}{"error": err.Error()})
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Use tracked total supply instead of recalculating
	currentSupply := t.totalSupply

	// Check maximum supply limit
	if t.maxSupply > 0 && currentSupply+amount > t.maxSupply {
		err := errors.New("mint amount would exceed maximum supply")
		log.Printf("Mint failed: %v (current: %d, requested: %d, max: %d)",
			err, currentSupply, amount, t.maxSupply)
		return err
	}

	// Overflow protection for recipient balance
	if t.balances[to] > ^uint64(0)-amount {
		err := errors.New("mint amount causes balance overflow")
		log.Printf("Mint failed: %v", err)
		return err
	}

	// Overflow protection for total supply
	if currentSupply > ^uint64(0)-amount {
		err := errors.New("mint amount causes total supply overflow")
		log.Printf("Mint failed: %v", err)
		return err
	}

	// Execute mint
	t.balances[to] += amount
	t.totalSupply += amount

	// Structured logging — pass balances directly to avoid re-acquiring the lock.
	LogTokenOperation(t, "mint", "", to, amount, "success", map[string]interface{}{
		"tx_hash":            t.generateTxHash("mint", to, amount),
		"to_balance_after":   t.balances[to],
		"total_supply_after": t.totalSupply,
	})

	// Emit event with enhanced metadata
	event := Event{
		Type:      EventMint,
		To:        to,
		Amount:    amount,
		Timestamp: time.Now(),
		TxHash:    t.generateTxHash("mint", to, amount),
		Metadata: map[string]interface{}{
			"new_balance":  t.balances[to],
			"total_supply": t.totalSupply,
			"max_supply":   t.maxSupply,
			"supply_ratio": float64(t.totalSupply) / float64(t.maxSupply),
		},
	}
	t.emitEvent(event)

	log.Printf("Mint successful: Balances[%s]=%d, TotalSupply=%d", to, t.balances[to], t.totalSupply)
	return nil
}

// ValidateSupply independently audits that the sum of all balances matches the tracked total supply
func (t *Token) ValidateSupply() (bool, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	sum := uint64(0)
	for _, balance := range t.balances {
		sum += balance
	}

	if sum != t.totalSupply {
		return false, fmt.Errorf("supply mismatch: tracked=%d, actual_sum=%d", t.totalSupply, sum)
	}
	return true, nil
}
