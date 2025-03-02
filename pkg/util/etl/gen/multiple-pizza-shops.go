package main

import (
    "fmt"
	"shadow-dom/etlctl/pkg/util/etl"
)

func main() {
	var etl etl.ETL = etl.ETL {
        Sources: []etl.DBStorage{
			
			{
				Name: "pizza_db",
				Type: "sqlite3",
				Connection: map[string]string{"filepath": `../etls/data/pizza.db`},
			},
			
			{
				Name: "pizza_2_db",
				Type: "sqlite3",
				Connection: map[string]string{"filepath": `../etls/data/pizza_2.db`},
			},
			
		},
		Queries: []etl.Query{
			
			{
				Name: "get_pizzas",
				SQL: `SELECT name, toppings FROM pizza;
`,
			},
			
		},
		Targets: []etl.DBStorage{
			
			{
				Name: "delivery_db",
				Type: "sqlite3",
				Connection: map[string]string{"filepath": `../etls/data/delivery.db`},
			},
			
		},
		Pipelines: []etl.Pipeline{
			
			{
				Sources: []string{ "pizza_db", "pizza_2_db" },
				Query:   "get_pizzas",
				Target:  "delivery_db.delivery",
				Fields: []etl.FieldMapping{
					{Source: "name", Target: "name"},{Source: "toppings", Target: "toppings"},
				},
			},
			
		},
	}

    fmt.Println(etl)
}
