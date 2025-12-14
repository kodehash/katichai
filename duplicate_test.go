package main

import "fmt"

// CalculateSum adds two numbers together
func CalculateSum(a, b int) int {
	return a + b
}

// AddNumbers adds two numbers together (duplicate of CalculateSum)
func AddNumbers(x, y int) int {
	return x + y
}

// MultiplyNumbers multiplies two numbers
func MultiplyNumbers(a, b int) int {
	return a * b
}

func main() {
	sum1 := CalculateSum(5, 3)
	sum2 := AddNumbers(10, 20)
	product := MultiplyNumbers(4, 5)
	
	fmt.Printf("Sum1: %d\n", sum1)
	fmt.Printf("Sum2: %d\n", sum2)
	fmt.Printf("Product: %d\n", product)
}
