package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	wallet "github.com/Shivam-Patel-G/blackhole-blockchain/services/wallet/wallet"
)

func checkTokenBalance(ctx context.Context, user *wallet.User) {
	fmt.Println("=== Check Token Balance ===")

	// Get wallet name
	fmt.Print("Enter wallet name: ")
	walletName := readLine()

	// Get password
	fmt.Print("Enter password: ")
	password := readLine()

	// Get token symbol
	fmt.Print("Enter token symbol (e.g., BHIV): ")
	tokenSymbol := readLine()

	balance, err := wallet.CheckTokenBalance(ctx, user, walletName, password, tokenSymbol)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Balance of %s: %d tokens\n", tokenSymbol, balance)
}

func transferTokens(ctx context.Context, user *wallet.User) {
	fmt.Println("=== Transfer Tokens ===")

	// Get wallet name
	fmt.Print("Enter your wallet name: ")
	walletName := readLine()

	// Get password
	fmt.Print("Enter password: ")
	password := readLine()

	// Get recipient address
	fmt.Print("Enter recipient address: ")
	toAddress := readLine()

	// Get token symbol
	fmt.Print("Enter token symbol (e.g., BHIV): ")
	tokenSymbol := readLine()

	// Get amount
	fmt.Print("Enter amount to transfer: ")
	amountStr := readLine()
	amount, err := strconv.ParseUint(amountStr, 10, 64)
	if err != nil {
		fmt.Printf("Invalid amount: %v\n", err)
		return
	}

	err = wallet.TransferTokensWithHistory(ctx, user, walletName, password, toAddress, tokenSymbol, amount)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Successfully transferred %d %s tokens to %s\n", amount, tokenSymbol, toAddress)
}

func stakeTokens(ctx context.Context, user *wallet.User) {
	fmt.Println("=== Stake Tokens ===")

	// Get wallet name
	fmt.Print("Enter your wallet name: ")
	walletName := readLine()

	// Get password
	fmt.Print("Enter password: ")
	password := readLine()

	// Get token symbol
	fmt.Print("Enter token symbol (e.g., BHIV): ")
	tokenSymbol := readLine()

	// Get amount
	fmt.Print("Enter amount to stake: ")
	amountStr := readLine()
	amount, err := strconv.ParseUint(amountStr, 10, 64)
	if err != nil {
		fmt.Printf("Invalid amount: %v\n", err)
		return
	}

	err = wallet.StakeTokensWithHistory(ctx, user, walletName, password, tokenSymbol, amount)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Successfully staked %d %s tokens\n", amount, tokenSymbol)
}

func importWalletFromPrivateKey(ctx context.Context, user *wallet.User) {
	fmt.Println("=== Import Wallet from Private Key ===")

	fmt.Print("Enter wallet name: ")
	walletName := readLine()

	fmt.Print("Enter password to secure the wallet: ")
	password := readLine()

	fmt.Print("Enter private key (hex): ")
	privateKeyHex := readLine()

	wallet, err := wallet.ImportWalletFromPrivateKey(ctx, user, password, walletName, privateKeyHex)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Successfully imported wallet: %s\n", wallet.WalletName)
	fmt.Printf("Address: %s\n", wallet.Address)
}

func exportWalletPrivateKey(ctx context.Context, user *wallet.User) {
	fmt.Println("=== Export Wallet Private Key ===")

	fmt.Print("Enter wallet name: ")
	walletName := readLine()

	fmt.Print("Enter password: ")
	password := readLine()

	privateKeyHex, err := wallet.ExportWalletPrivateKey(ctx, user, walletName, password)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Private Key: %s\n", privateKeyHex)
	fmt.Println("⚠️ Keep this private key secure and never share it!")
}

func viewTransactionHistory(ctx context.Context, user *wallet.User) {
	fmt.Println("=== Transaction History ===")

	fmt.Print("Enter wallet address (or press Enter for all transactions): ")
	walletAddr := readLine()

	var transactions []*wallet.TransactionRecord
	var err error

	if walletAddr == "" {
		transactions, err = wallet.GetAllUserTransactions(ctx, user.ID.Hex(), 50)
	} else {
		transactions, err = wallet.GetWalletTransactionHistory(ctx, user.ID.Hex(), walletAddr, 50)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(transactions) == 0 {
		fmt.Println("No transactions found.")
		return
	}

	fmt.Printf("Found %d transactions:\n\n", len(transactions))
	for i, tx := range transactions {
		fmt.Printf("%d. %s\n", i+1, tx.Type)
		fmt.Printf("   From: %s\n", tx.From)
		fmt.Printf("   To: %s\n", tx.To)
		fmt.Printf("   Amount: %d %s\n", tx.Amount, tx.TokenSymbol)
		fmt.Printf("   Status: %s\n", tx.Status)
		fmt.Printf("   Time: %s\n", tx.Timestamp.Format(time.RFC3339))
		if tx.BlockHeight > 0 {
			fmt.Printf("   Block: %d\n", tx.BlockHeight)
		}
		fmt.Println()
	}
}

func listAllWallets(ctx context.Context, user *wallet.User) {
	fmt.Println("=== All User Wallets ===")

	wallets, err := wallet.ListUserWallets(ctx, user)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(wallets) == 0 {
		fmt.Println("No wallets found.")
		return
	}

	fmt.Printf("Found %d wallets:\n\n", len(wallets))
	for i, w := range wallets {
		fmt.Printf("%d. %s\n", i+1, w.WalletName)
		fmt.Printf("   Address: %s\n", w.Address)
		fmt.Printf("   Created: %s\n", w.CreatedAt.Format(time.RFC3339))
		fmt.Println()
	}
}

// Web server functionality
var sessions = make(map[string]*SessionData)

type SessionData struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
