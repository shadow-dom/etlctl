/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"shadow-dom/etlctl/pkg/builder"
	"shadow-dom/etlctl/pkg/util/etl"
)

// "shadow-dom/etlctl/pkg/cmd"

func main() {
	// cmd.Execute()
	etl.Run("multiple-pizza-shops")
	builder.GenerateETL("multiple-pizza-shops")
}
