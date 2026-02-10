package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "http://TCG-USS-AE:7001/tcg-uss-ae/customer?customerId=42755"

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
