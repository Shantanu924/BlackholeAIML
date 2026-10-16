package token

import (
	"fmt"
	"time"
)

// VestingSchedule defines a token vesting schedule for a specific address
type VestingSchedule struct {
	Beneficiary  string
	TotalAmount  uint64
	Released     uint64
	StartTime    time.Time
	CliffDuration time.Duration
	Duration     time.Duration
	Revocable    bool
	Revoked      bool
}

// Tokenomics handles advanced economic logic like fee burning, distribution, and vesting
type Tokenomics struct {
	Token            *Token
	VestingSchedules map[string]*VestingSchedule
	BaseFeeBurnRate  float64 // Percentage of fee to burn (e.g., 0.50 for 50%)
	TreasuryFeeRate  float64 // Percentage of fee to treasury (e.g., 0.50 for 50%)
}

// NewTokenomics creates a new tokenomics module for a token
func NewTokenomics(token *Token) *Tokenomics {
	return &Tokenomics{
		Token:            token,
		VestingSchedules: make(map[string]*VestingSchedule),
		BaseFeeBurnRate:  0.50, // Default 50% burn
		TreasuryFeeRate:  0.50, // Default 50% to treasury
	}
}

// CreateVestingSchedule sets up a new vesting schedule for a beneficiary
func (tk *Tokenomics) CreateVestingSchedule(beneficiary string, amount uint64, startTime time.Time, cliff time.Duration, duration time.Duration, revocable bool) error {
	tk.Token.mu.Lock()
	defer tk.Token.mu.Unlock()

	if beneficiary == "" {
		return fmt.Errorf("invalid beneficiary address")
	}
	if amount == 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	tk.VestingSchedules[beneficiary] = &VestingSchedule{
		Beneficiary:  beneficiary,
		TotalAmount:  amount,
		Released:     0,
		StartTime:    startTime,
		CliffDuration: cliff,
		Duration:     duration,
		Revocable:    revocable,
		Revoked:      false,
	}

	return nil
}

// VestedAmount calculates the currently vested amount for a schedule
func (tk *Tokenomics) VestedAmount(schedule *VestingSchedule, timestamp time.Time) uint64 {
	if timestamp.Before(schedule.StartTime.Add(schedule.CliffDuration)) {
		return 0
	}
	if timestamp.After(schedule.StartTime.Add(schedule.Duration)) || schedule.Revoked {
		return schedule.TotalAmount
	}

	passed := timestamp.Sub(schedule.StartTime)
	return uint64((float64(schedule.TotalAmount) * float64(passed)) / float64(schedule.Duration))
}

// ReleaseVested releases available vested tokens to the beneficiary
func (tk *Tokenomics) ReleaseVested(beneficiary string) (uint64, error) {
	tk.Token.mu.Lock()
	defer tk.Token.mu.Unlock()

	schedule, exists := tk.VestingSchedules[beneficiary]
	if !exists {
		return 0, fmt.Errorf("no vesting schedule found for %s", beneficiary)
	}

	vested := tk.VestedAmount(schedule, time.Now())
	unreleased := vested - schedule.Released

	if unreleased <= 0 {
		return 0, fmt.Errorf("no tokens due for release")
	}

	schedule.Released += unreleased
	
	// Mint or transfer the unreleased amount to the beneficiary
	tk.Token.balances[beneficiary] += unreleased
	tk.Token.totalSupply += unreleased

	txHash := tk.Token.generateTxHash("VESTING_RELEASE", beneficiary, unreleased)
	tk.Token.events = append(tk.Token.events, Event{
		Type:      "VestingRelease",
		From:      "system",
		To:        beneficiary,
		Amount:    unreleased,
		Timestamp: time.Now(),
		TxHash:    txHash,
	})

	return unreleased, nil
}

// ProcessTransactionFee handles the burning and distribution of a transaction fee
func (tk *Tokenomics) ProcessTransactionFee(sender string, totalFee uint64) error {
	tk.Token.mu.Lock()
	defer tk.Token.mu.Unlock()

	if totalFee == 0 {
		return nil
	}

	burnAmount := uint64(float64(totalFee) * tk.BaseFeeBurnRate)
	treasuryAmount := totalFee - burnAmount

	// Ensure sender has enough balance
	if tk.Token.balances[sender] < totalFee {
		return fmt.Errorf("insufficient balance for transaction fee")
	}

	// Deduct total fee
	tk.Token.balances[sender] -= totalFee

	// Process Burn
	if burnAmount > 0 {
		tk.Token.totalSupply -= burnAmount
		txHash := tk.Token.generateTxHash("FEE_BURN", sender, burnAmount)
		tk.Token.events = append(tk.Token.events, Event{
			Type:      "FeeBurn",
			From:      sender,
			To:        "0x0000000000000000000000000000000000000000",
			Amount:    burnAmount,
			Timestamp: time.Now(),
			TxHash:    txHash,
		})
	}

	// Process Treasury
	if treasuryAmount > 0 {
		tk.Token.balances[tk.Token.treasuryAddress] += treasuryAmount
		txHash := tk.Token.generateTxHash("FEE_TREASURY", sender, treasuryAmount)
		tk.Token.events = append(tk.Token.events, Event{
			Type:      "FeeTreasury",
			From:      sender,
			To:        tk.Token.treasuryAddress,
			Amount:    treasuryAmount,
			Timestamp: time.Now(),
			TxHash:    txHash,
		})
	}

	return nil
}
