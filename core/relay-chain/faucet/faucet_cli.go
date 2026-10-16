package faucet

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// FaucetCLI provides a command-line interface for the validator faucet
type FaucetCLI struct {
	faucet *ValidatorFaucet
	reader *bufio.Reader
}

// NewFaucetCLI creates a new faucet CLI
func NewFaucetCLI(faucet *ValidatorFaucet) *FaucetCLI {
	return &FaucetCLI{
		faucet: faucet,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Start starts the interactive CLI
func (cli *FaucetCLI) Start() {
	fmt.Println("🚰 Blackhole Validator Faucet CLI")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println()
	
	for {
		cli.showMenu()
		choice := cli.readInput("Enter your choice: ")
		
		switch strings.TrimSpace(choice) {
		case "1":
			cli.showStats()
		case "2":
			cli.listValidators()
		case "3":
			cli.registerValidator()
		case "4":
			cli.distributeTokens()
		case "5":
			cli.showValidatorInfo()
		case "6":
			cli.showDistributionHistory()
		case "7":
			cli.configureSettings()
		case "8":
			cli.emergencyDistribution()
		case "9":
			fmt.Println("👋 Goodbye!")
			return
		default:
			fmt.Println("❌ Invalid choice. Please try again.")
		}
		
		fmt.Println()
	}
}

// showMenu displays the main menu
func (cli *FaucetCLI) showMenu() {
	fmt.Println("📋 Main Menu:")
	fmt.Println("  1. Show Faucet Statistics")
	fmt.Println("  2. List All Validators")
	fmt.Println("  3. Register New Validator")
	fmt.Println("  4. Distribute Tokens")
	fmt.Println("  5. Show Validator Info")
	fmt.Println("  6. Show Distribution History")
	fmt.Println("  7. Configure Settings")
	fmt.Println("  8. Emergency Distribution")
	fmt.Println("  9. Exit")
	fmt.Println()
}

// showStats displays faucet statistics
func (cli *FaucetCLI) showStats() {
	fmt.Println("📊 Faucet Statistics")
	fmt.Println("-" + strings.Repeat("-", 30))
	
	stats := cli.faucet.GetFaucetStats()
	
	fmt.Printf("💰 Total Distributed: %d BHIV\n", stats.TotalDistributed)
	fmt.Printf("👥 Total Validators: %d\n", stats.TotalValidators)
	fmt.Printf("✅ Active Validators: %d\n", stats.ActiveValidators)
	fmt.Printf("📅 Distributions Today: %d BHIV\n", stats.DistributionsToday)
	fmt.Printf("⏰ Last Distribution: %s\n", formatTime(stats.LastDistribution))
	fmt.Printf("🚀 Uptime: %s\n", time.Since(stats.StartTime).Round(time.Second))
}

// listValidators displays all registered validators
func (cli *FaucetCLI) listValidators() {
	fmt.Println("👥 Registered Validators")
	fmt.Println("-" + strings.Repeat("-", 50))
	
	validators := cli.faucet.ListValidators()
	
	if len(validators) == 0 {
		fmt.Println("No validators registered.")
		return
	}
	
	fmt.Printf("%-20s %-8s %-12s %-15s\n", "Address", "Active", "Stake", "Total Received")
	fmt.Println(strings.Repeat("-", 60))
	
	for address, info := range validators {
		status := "❌"
		if info.IsActive {
			status = "✅"
		}
		
		shortAddr := address
		if len(address) > 18 {
			shortAddr = address[:15] + "..."
		}
		
		fmt.Printf("%-20s %-8s %-12d %-15d\n", 
			shortAddr, status, info.TotalStake, info.TotalReceived)
	}
}

// registerValidator registers a new validator
func (cli *FaucetCLI) registerValidator() {
	fmt.Println("📝 Register New Validator")
	fmt.Println("-" + strings.Repeat("-", 30))
	
	address := cli.readInput("Enter validator address: ")
	address = strings.TrimSpace(address)
	
	if address == "" {
		fmt.Println("❌ Address cannot be empty.")
		return
	}
	
	err := cli.faucet.RegisterValidator(address)
	if err != nil {
		fmt.Printf("❌ Failed to register validator: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Validator %s registered successfully!\n", address)
}

// distributeTokens manually distributes tokens to a validator
func (cli *FaucetCLI) distributeTokens() {
	fmt.Println("💰 Distribute Tokens")
	fmt.Println("-" + strings.Repeat("-", 25))
	
	address := cli.readInput("Enter validator address: ")
	address = strings.TrimSpace(address)
	
	if address == "" {
		fmt.Println("❌ Address cannot be empty.")
		return
	}
	
	amountStr := cli.readInput("Enter amount to distribute: ")
	amount, err := strconv.ParseUint(strings.TrimSpace(amountStr), 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid amount: %v\n", err)
		return
	}
	
	reason := cli.readInput("Enter reason (optional): ")
	if strings.TrimSpace(reason) == "" {
		reason = ReasonManual
	}
	
	err = cli.faucet.DistributeToValidator(address, amount, reason)
	if err != nil {
		fmt.Printf("❌ Failed to distribute tokens: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Successfully distributed %d BHIV to %s\n", amount, address)
}

// showValidatorInfo displays detailed information about a validator
func (cli *FaucetCLI) showValidatorInfo() {
	fmt.Println("ℹ️ Validator Information")
	fmt.Println("-" + strings.Repeat("-", 30))
	
	address := cli.readInput("Enter validator address: ")
	address = strings.TrimSpace(address)
	
	if address == "" {
		fmt.Println("❌ Address cannot be empty.")
		return
	}
	
	info, err := cli.faucet.GetValidatorInfo(address)
	if err != nil {
		fmt.Printf("❌ Failed to get validator info: %v\n", err)
		return
	}
	
	fmt.Printf("📍 Address: %s\n", info.Address)
	fmt.Printf("✅ Active: %t\n", info.IsActive)
	fmt.Printf("💰 Total Stake: %d BHIV\n", info.TotalStake)
	fmt.Printf("🎁 Total Received: %d BHIV\n", info.TotalReceived)
	fmt.Printf("🏗️ Blocks Produced: %d\n", info.BlocksProduced)
	fmt.Printf("👀 First Seen: %s\n", formatTime(info.FirstSeen))
	fmt.Printf("👁️ Last Seen: %s\n", formatTime(info.LastSeen))
	fmt.Printf("💸 Last Distribution: %s\n", formatTime(info.LastDistribution))
}

// showDistributionHistory displays distribution history for a validator
func (cli *FaucetCLI) showDistributionHistory() {
	fmt.Println("📜 Distribution History")
	fmt.Println("-" + strings.Repeat("-", 30))
	
	address := cli.readInput("Enter validator address: ")
	address = strings.TrimSpace(address)
	
	if address == "" {
		fmt.Println("❌ Address cannot be empty.")
		return
	}
	
	limitStr := cli.readInput("Enter limit (default 10): ")
	limit := 10
	if strings.TrimSpace(limitStr) != "" {
		if l, err := strconv.Atoi(strings.TrimSpace(limitStr)); err == nil {
			limit = l
		}
	}
	
	history := cli.faucet.GetDistributionHistory(address, limit)
	
	if len(history) == 0 {
		fmt.Println("No distribution history found.")
		return
	}
	
	fmt.Printf("%-12s %-15s %-20s %-10s\n", "Amount", "Reason", "Timestamp", "TX Hash")
	fmt.Println(strings.Repeat("-", 70))
	
	for _, record := range history {
		shortHash := record.TxHash
		if len(shortHash) > 8 {
			shortHash = shortHash[:8] + "..."
		}
		
		fmt.Printf("%-12d %-15s %-20s %-10s\n",
			record.Amount,
			record.Reason,
			record.Timestamp.Format("2006-01-02 15:04"),
			shortHash)
	}
}

// configureSettings allows configuration of faucet settings
func (cli *FaucetCLI) configureSettings() {
	fmt.Println("⚙️ Configure Settings")
	fmt.Println("-" + strings.Repeat("-", 25))
	
	fmt.Println("Current Configuration:")
	config := cli.faucet.config
	fmt.Printf("  Initial Amount: %d BHIV\n", config.InitialValidatorAmount)
	fmt.Printf("  Top-up Amount: %d BHIV\n", config.TopUpAmount)
	fmt.Printf("  Top-up Threshold: %d BHIV\n", config.TopUpThreshold)
	fmt.Printf("  Cooldown Period: %v\n", config.CooldownPeriod)
	fmt.Printf("  Daily Limit: %d BHIV\n", config.MaxDailyDistribution)
	fmt.Printf("  Auto Distribution: %t\n", config.AutoDistributionEnabled)
	fmt.Println()
	
	fmt.Println("Note: Configuration changes require restart to take effect.")
}

// emergencyDistribution performs emergency token distribution
func (cli *FaucetCLI) emergencyDistribution() {
	fmt.Println("🚨 Emergency Distribution")
	fmt.Println("-" + strings.Repeat("-", 30))
	
	fmt.Println("⚠️ Warning: This bypasses normal eligibility checks!")
	confirm := cli.readInput("Are you sure? (yes/no): ")
	
	if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
		fmt.Println("❌ Emergency distribution cancelled.")
		return
	}
	
	address := cli.readInput("Enter validator address: ")
	address = strings.TrimSpace(address)
	
	if address == "" {
		fmt.Println("❌ Address cannot be empty.")
		return
	}
	
	amountStr := cli.readInput("Enter emergency amount: ")
	amount, err := strconv.ParseUint(strings.TrimSpace(amountStr), 10, 64)
	if err != nil {
		fmt.Printf("❌ Invalid amount: %v\n", err)
		return
	}
	
	// Bypass normal checks for emergency distribution
	err = cli.faucet.tokenSystem.Mint(address, amount)
	if err != nil {
		fmt.Printf("❌ Failed to mint emergency tokens: %v\n", err)
		return
	}
	
	fmt.Printf("🚨 Emergency distribution completed: %d BHIV to %s\n", amount, address)
}

// Helper methods

// readInput reads a line of input from the user
func (cli *FaucetCLI) readInput(prompt string) string {
	fmt.Print(prompt)
	input, _ := cli.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}
