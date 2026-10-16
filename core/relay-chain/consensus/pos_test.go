package consensus

import (
	"testing"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/chain"
)

func TestValidator_SelectValidator_EmptyStakes(t *testing.T) {
	sl := &chain.StakeLedger{
		Stakes: make(map[string]uint64),
	}
	v := NewValidator(sl)

	chosen := v.SelectValidator()
	if chosen != "" {
		t.Fatalf("expected empty string for zero stakes, got %q", chosen)
	}
}

func TestValidator_SelectValidator_SingleStake(t *testing.T) {
	sl := &chain.StakeLedger{
		Stakes: make(map[string]uint64),
	}
	sl.SetStake("validator-alpha", 5000)
	v := NewValidator(sl)

	chosen := v.SelectValidator()
	if chosen != "validator-alpha" {
		t.Fatalf("expected 'validator-alpha', got %q", chosen)
	}
}

func TestValidator_SelectValidator_MultipleStakes(t *testing.T) {
	sl := &chain.StakeLedger{
		Stakes: make(map[string]uint64),
	}
	sl.SetStake("validator-alpha", 1000)
	sl.SetStake("validator-beta", 2000)
	v := NewValidator(sl)

	chosen := v.SelectValidator()
	if chosen != "validator-alpha" && chosen != "validator-beta" {
		t.Fatalf("expected either validator-alpha or validator-beta, got %q", chosen)
	}
}

func TestDynamicRewardStrategy(t *testing.T) {
	baseReward := uint64(100)
	maxSupply := uint64(1_000_000)
	minReward := uint64(20)

	strategy := NewDynamicRewardStrategy(baseReward, maxSupply, minReward)

	// Case 1: Early supply (20% of max supply) -> full base reward
	rewardEarly := strategy.CalculateReward(200_000)
	if rewardEarly != baseReward {
		t.Fatalf("expected base reward %d at 20%% supply, got %d", baseReward, rewardEarly)
	}

	// Case 2: High supply (80% of max supply) -> reduced reward
	rewardLate := strategy.CalculateReward(800_000)
	if rewardLate >= baseReward {
		t.Fatalf("expected reward to be reduced below %d at 80%% supply, got %d", baseReward, rewardLate)
	}

	// Case 3: Exhausted supply (100% of max supply) -> min reward constraint
	rewardExhausted := strategy.CalculateReward(1_000_000)
	if rewardExhausted < minReward {
		t.Fatalf("expected reward >= minReward %d, got %d", minReward, rewardExhausted)
	}

	// Case 4: Reward Info metadata
	info := strategy.GetRewardInfo(800_000)
	if info["reduction_active"] != true {
		t.Fatalf("expected reduction_active to be true at 80%% supply")
	}
}

func TestValidator_IntervalValidation(t *testing.T) {
	sl := chain.NewStakeLedger()
	v := NewValidator(sl)
	v.BlockInterval = 5 * time.Second
	v.LastBlockTime = time.Now() // block mined just now

	dummyBlock := &chain.Block{
		Header: chain.BlockHeader{
			Index: 2,
		},
	}
	bc := &chain.Blockchain{}

	// Validation must fail because not enough time has elapsed
	valid := v.ValidateBlock(dummyBlock, bc)
	if valid {
		t.Fatalf("expected ValidateBlock to reject block mined too early")
	}
}
