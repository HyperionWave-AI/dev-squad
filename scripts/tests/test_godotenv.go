package main

import (
	"fmt"
	"os"
	"github.com/joho/godotenv"
)

func main() {
	// Test loading ../bin/.env.hyper
	err := godotenv.Overload("../bin/.env.hyper")
	if err != nil {
		fmt.Printf("Error loading bin/.env.hyper: %v\n", err)
	} else {
		fmt.Println("✓ Successfully loaded bin/.env.hyper")
		fmt.Printf("MONGODB_URI: %s...\n", os.Getenv("MONGODB_URI")[:50])
	}
}
