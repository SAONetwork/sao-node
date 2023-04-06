package types

import (
	"fmt"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
)

type SidInfo struct {
	Did            string
	PaymentAddress string
	Accounts       []Account
	PastSeeds      []string
	DidDocuments   []DidDocument
}

type KidInfo struct {
	Did            string
	PaymentAddress string
	Document       saodidtypes.DidDocument
}

type Account struct {
	AccountId            string
	AccountDid           string
	AccountEncryptedSeed string
	SidEncryptedAccount  string
}

type DidDocument struct {
	Version  string
	Document saodidtypes.DidDocument
}

type DidInfo interface {
	PrintInfo()
}

func (si SidInfo) PrintInfo() {
	fmt.Println("Did: ", si.Did)
	fmt.Println("PaymentAddress:", si.PaymentAddress)
	fmt.Println("Accounts:")
	for index, account := range si.Accounts {
		fmt.Println("  Account", index, " id: ", account.AccountId)
		fmt.Println("    AccountDid: ", account.AccountDid)
		fmt.Println("    AccountEncryptedSeed: ", account.AccountEncryptedSeed)
		fmt.Println("    SidEncryptedAccount:  ", account.SidEncryptedAccount)
	}
	fmt.Println()

	if len(si.PastSeeds) != 0 {
		printStringArray(si.PastSeeds, "PastSeeds", "")
		fmt.Println()
	}

	fmt.Println("DidDocument:")
	for index, document := range si.DidDocuments {
		fmt.Println("  DocId", index, ": ", document.Version)
		printDidDocument(document.Document, "    ")
	}
	fmt.Println()
}

func (ki KidInfo) PrintInfo() {
	fmt.Println("Did: ", ki.Did)
	fmt.Println("PaymentAddress:", ki.PaymentAddress)
	fmt.Println("DidDocument:")
	printDidDocument(ki.Document, "  ")
}

func printDidDocument(didDocument saodidtypes.DidDocument, prefix string) {
	printVm := func(vm saodidtypes.VerificationMethod) {
		fmt.Println(prefix+"  Id: ", vm.Id)
		fmt.Println(prefix+"    Type:            ", vm.Type)
		fmt.Println(prefix+"    Controller:      ", vm.Controller)
		fmt.Println(prefix+"    PublicKeyBase58: ", vm.PublicKeyBase58)
	}

	// context
	printStringArray(didDocument.Context, "Context", prefix)

	// id
	fmt.Println(prefix+"Id: ", didDocument.Id)

	// also known as
	printStringArray(didDocument.AlsoKnownAs, "AlsoKnownAs", prefix)

	// controller
	printStringArray(didDocument.Controller, "Controller", prefix)

	// verification method
	if len(didDocument.VerificationMethod) > 0 {
		fmt.Println(prefix + "VerificationMethods: ")
		for _, vm := range didDocument.VerificationMethod {
			printVm(vm)
		}
	}

	// authentication
	if len(didDocument.Authentication) > 0 {
		fmt.Println(prefix + "Authentication: ")
		for _, vmany := range didDocument.Authentication {
			switch t := vmany.(type) {
			case string:
				fmt.Println(prefix + "- " + t)
			case saodidtypes.VerificationMethod:
				printVm(t)
			}
		}
	}

	// key agreement
	if len(didDocument.KeyAgreement) > 0 {
		fmt.Println(prefix + "KeyAgreement: ")
		for _, vm := range didDocument.KeyAgreement {
			printVm(vm)
		}
	}

}

func printStringArray(array []string, name, prefix string) {
	if len(array) > 0 {
		fmt.Println(prefix + name + ": [")
		for _, controller := range array {
			fmt.Println(prefix + "  " + controller)
		}
		fmt.Println(prefix + "]")
	}
}
