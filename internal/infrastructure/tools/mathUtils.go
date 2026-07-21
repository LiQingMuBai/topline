package tools

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

func SubtractAndRound(s1, s2 string, n float64) (string, error) {
	// Step 1: Parse strings to float64
	a, err := strconv.ParseFloat(s1, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number s1: %w", err)
	}
	b, err := strconv.ParseFloat(s2, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number s2: %w", err)
	}

	// Step 2: Subtract
	result := a - b*n

	// Step 3: Round to 1 decimal place (round half up)
	// Multiply by 10, round, then divide by 10
	rounded := math.Round(result*10) / 10

	//return rounded, nil

	// 3. 将结果转为字符串
	return fmt.Sprintf("%v", rounded), nil

}

func SubtractStringNumbers(a, b string, n float64) (string, error) {
	// 1. 将字符串转为 float64
	numA, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return "", fmt.Errorf("转换 %s 失败: %v", a, err)
	}

	numB, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return "", fmt.Errorf("转换 %s 失败: %v", b, err)
	}

	// 2. 计算减法
	result := numA - numB*n
	rounded := math.Round(result*10) / 10
	// 3. 将结果转为字符串
	return fmt.Sprintf("%v", rounded), nil
}
func CompareStringsWithFloat(a, b string, n float64) bool {
	// 将字符串转换为 float64
	floatA, errA := strconv.ParseFloat(a, 64)
	floatB, errB := strconv.ParseFloat(b, 64)

	if errA != nil || errB != nil {
		return false
	}

	// 计算 b * 2
	bTimesTwo := floatB * n

	// 比较 a 和 b * 2
	return floatA >= bTimesTwo
}
func StringMultiply(s string, n float64) (string, error) {
	// 将字符串转换为 int64
	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return "", fmt.Errorf("无法将字符串转换为int64: %v", err)
	}

	// 执行乘法运算
	result := num * n

	// 将结果转换回字符串
	return fmt.Sprintf("%v", result), nil
}

func StringMultiply2(s string, n float64) string {
	// 将字符串转换为 int64
	num, _ := strconv.ParseFloat(s, 64)

	// 执行乘法运算
	result := num * n

	// 将结果转换回字符串
	return fmt.Sprintf("%v", result)
}

func AddStringsAsFloats(a, b string) string {
	// 1. 将第一个字符串转换成 float64
	num1, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return "0"
	}

	// 2. 将第二个字符串转换成 float64
	num2, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return "0"
	}

	// 3. 相加并返回结果

	sum := num1 + num2
	amount := fmt.Sprintf("%f", sum)

	return amount[0 : len(amount)-3]
}
func Generate6DigitOrderNo() string {
	// 获取当前时间的秒数(0-59)和纳秒的后4位
	now := time.Now()
	seconds := now.Second()           // 0-59
	nanos := now.Nanosecond() % 10000 // 取纳秒的后4位

	// 组合成6位数: 秒数(2位) + 纳秒后4位
	return fmt.Sprintf("%02d%04d", seconds, nanos/100)
}

func CompareNumberStrings(a, b string) (int, error) {
	numA, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number string: %s", a)
	}

	numB, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number string: %s", b)
	}

	if numA < numB {
		return -1, nil
	} else if numA > numB {
		return 1, nil
	}
	return 0, nil
}
