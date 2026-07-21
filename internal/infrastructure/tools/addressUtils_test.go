package tools

import (
	"fmt"
	"testing"
)

func TestIsValidBitcoinAddress(t *testing.T) {
	//1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
	//bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4
	//0xF510e53EF8DA4e45FFA59EB554511a7410E5eFD3
	valid := IsValidBitcoinAddress("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")

	fmt.Printf("valid: %v\n", valid)
}
