package main

import "fmt"

// VeryLongFunctionNameThatDoesTooManyThings is an example of a function
// that would trigger code quality warnings
func VeryLongFunctionNameThatDoesTooManyThings(a, b, c, d, e, f, g int) int {
	result := 0
	
	// Lots of complexity
	if a > 0 {
		if b > 0 {
			if c > 0 {
				if d > 0 {
					if e > 0 {
						if f > 0 {
							if g > 0 {
								result = a + b + c + d + e + f + g
							}
						}
					}
				}
			}
		}
	}
	
	// More lines to make it long
	for i := 0; i < 100; i++ {
		result += i
		if i%2 == 0 {
			result *= 2
		} else {
			result /= 2
		}
		
		if result > 1000 {
			break
		}
		
		if result < 0 {
			continue
		}
		
		fmt.Println(result)
	}
	
	return result
}

func main() {
	result := VeryLongFunctionNameThatDoesTooManyThings(1, 2, 3, 4, 5, 6, 7)
	fmt.Println("Result:", result)
}
