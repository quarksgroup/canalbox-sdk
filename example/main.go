package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/quarksgroup/canalbox-sdk"
)

func main() {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	username := os.Getenv("CANALBOX_USERNAME")
	password := os.Getenv("CANALBOX_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("CANALBOX_USERNAME and CANALBOX_PASSWORD are required")
	}

	client, err := canalbox.Login(
		username,
		password,
		nil,
	)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	fmt.Println("Login successful!")

	account, err := client.GetAccountDetails(ctx)
	if err != nil {
		fmt.Printf("Could not load account balance: %v\n", err)
	} else {
		fmt.Printf("Current account balance: %s\n", strconv.FormatFloat(account.Balance, 'f', -1, 64))
	}

	boxNumber := promptRequired(reader, "Enter box number")
	sub, err := client.GetSubscriptionByBoxNumber(boxNumber)
	if err != nil {
		log.Fatalf("failed to get subscription by box: %v", err)
	}

	fmt.Printf("\nSubscription: %s\n", sub.Name)
	fmt.Printf("  Box Number: %s\n", sub.BoxNumber__c)
	fmt.Printf("  Product: %s\n", sub.SUB_T_Produit_De_Base__c)
	fmt.Printf("  Account: %s\n", sub.Zuora__Account__r.Name)
	fmt.Printf("  Phone: %s\n", sub.Zuora__Account__r.Phone)
	fmt.Printf("  Renewal Date: %s\n", sub.ExpectedRenewalDate__c)

	options, err := client.GetRenewOptionsByBox(ctx, boxNumber)
	if err != nil {
		log.Fatalf("failed to load renewal options: %v", err)
	}

	offers := options.Offers
	if len(offers) == 0 {
		log.Fatal("no offers available for this subscription")
	}

	fmt.Println("\nAvailable offers:")
	for i, offer := range offers {
		label := ""
		if offer.Current {
			label = " (current)"
		}
		fmt.Printf("  %d) %s%s\n", i+1, offer.Name, label)
	}

	offerIndex := promptIntInRange(reader, "Choose offer number", 1, len(offers))
	chosenOffer := offers[offerIndex-1]
	months := promptIntInRange(reader, "Choose number of months", 1, 24)

	preview, err := client.PreviewRenewByBox(ctx, boxNumber, chosenOffer.Name, months)
	if err != nil {
		log.Fatalf("preview failed: %v", err)
	}

	if !preview.Success {
		fmt.Println("\nPreview blocked:")
		for _, reason := range preview.Reasons {
			fmt.Println(" -", reason.Message)
		}
		return
	}

	if preview.PreviewResult == nil || len(preview.PreviewResult.Invoices) == 0 {
		log.Fatal("preview succeeded but no invoice returned")
	}

	invoice := preview.PreviewResult.Invoices[0]
	fmt.Println("\nAmount to pay:")
	fmt.Printf("  Total: %.2f\n", invoice.Amount)
	fmt.Printf("  Before tax: %.2f\n", invoice.AmountWithoutTax)
	fmt.Printf("  Tax: %.2f\n", invoice.TaxAmount)
	fmt.Printf("  Target date: %s\n", invoice.TargetDate)

	if !promptYesNo(reader, "Proceed with activation", false) {
		fmt.Println("Activation cancelled.")
		return
	}

	activation, err := client.ActivateRenewByBox(ctx, boxNumber, chosenOffer.Name, months)
	if err != nil {
		log.Fatalf("activation failed: %v", err)
	}

	if !activation.Success {
		fmt.Println("Activation blocked:")
		for _, reason := range activation.Reasons {
			fmt.Println(" -", reason.Message)
		}
		return
	}

	fmt.Println("Subscription activated.")
}

func promptRequired(reader *bufio.Reader, label string) string {
	for {
		fmt.Printf("%s: ", label)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			return input
		}
		fmt.Println("Value is required.")
	}
}

func promptIntInRange(reader *bufio.Reader, label string, min, max int) int {
	for {
		fmt.Printf("%s (%d-%d): ", label, min, max)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		v, err := strconv.Atoi(input)
		if err == nil && v >= min && v <= max {
			return v
		}
		fmt.Printf("Please enter a number between %d and %d.\n", min, max)
	}
}

func promptYesNo(reader *bufio.Reader, label string, defaultYes bool) bool {
	defaultLabel := "y/N"
	if defaultYes {
		defaultLabel = "Y/n"
	}

	for {
		fmt.Printf("%s [%s]: ", label, defaultLabel)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))
		switch input {
		case "":
			return defaultYes
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
		fmt.Println("Please answer yes or no.")
	}
}
