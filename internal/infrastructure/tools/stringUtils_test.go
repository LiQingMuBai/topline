package tools

import (
	"fmt"
	"testing"
)

func TestExtractNumberBeforeBi(t *testing.T) {

	nums, err := ExtractNumberBeforeBi("5笔（15TRX）")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(nums)
}
