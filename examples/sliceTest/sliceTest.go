package main

import "fmt"

func reverse(s []int) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func main() {
	months := []string{1: "January", 2: "Feburary", 3: "March", 4: "April", 5: "March", 6: "June", 7: "July", 8: "August",
		9: "September", 10: "October", 11: "November", 12: "December"}

	fmt.Println(months)
	fmt.Println(len(months))
	fmt.Println(cap(months))

	Q2 := months[4:7]
	fmt.Println(Q2)
	fmt.Println(len(Q2))
	fmt.Println(cap(Q2))
	endlessQ2 := Q2[:9]
	fmt.Println(endlessQ2)
	summer := months[6:9]
	fmt.Println(summer)
	fmt.Println(len(summer))
	fmt.Println(cap(summer))
	endlessSummer := summer[:7]
	fmt.Println(endlessSummer)

}
