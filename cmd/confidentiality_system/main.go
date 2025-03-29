package main

import (
	"fmt"
	"log"
	"time"

	"github.com/danielbahrami/se10-mt/internal/postgres"
)

func main() {
	dbpool, err := postgres.ConnectPostgres()
	if err != nil {
		log.Fatalf("Error connecting to Postgres: %v", err)
	}
	defer dbpool.Close()

	i := 1
	for {
		fmt.Println(i)
		i++
		time.Sleep(10 * time.Second)
	}
}
