package main

import (
	"fmt"
	"log"

	"github.com/quarksgroup/canalbox-sdk"
)

func main() {
	client, err := canalbox.Login(
		"https://grpvivendiafrica.my.site.com",
		"you@email.com",
		"yourpassword",
	)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	fmt.Println("Login successful!")

	subs, err := client.GetSubscription("D6D306D4")
	if err != nil {
		log.Fatalf("Failed to get subscription: %v", err)
	}

	for _, sub := range subs {
		fmt.Printf("\nSubscription: %s\n", sub.Name)
		fmt.Printf("  Box Number: %s\n", sub.BoxNumber__c)
		fmt.Printf("  Product: %s\n", sub.SUB_T_Produit_De_Base__c)
		fmt.Printf("  Account: %s\n", sub.Zuora__Account__r.Name)
		fmt.Printf("  Phone: %s\n", sub.Zuora__Account__r.Phone)
		fmt.Printf("  Renewal Date: %s\n", sub.ExpectedRenewalDate__c)
	}
}
