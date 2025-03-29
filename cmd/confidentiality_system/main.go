package main

import (
	"fmt"
	"time"

	"github.com/danielbahrami/se10-mt/internal/postgres"
)

func main() {
	postgres.ConnectPostgres()

	i := 1
	for {
		fmt.Println(i)
		i++
		time.Sleep(10 * time.Second)
	}
}
