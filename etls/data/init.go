package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

const SCHEMA = `
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	toppings TEXT NOT NULL
`

func initDB(name string, insertDummyData bool) {
	fileName := name + ".db"

	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		db, err := sql.Open("sqlite3", fileName)
		if err != nil {
			log.Fatalf("Failed to open %s: %v", fileName, err)
		}
		defer db.Close()

		sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", name, SCHEMA)

		_, err = db.Exec(sql)
		if err != nil {
			log.Fatalf("Failed to create %s table: %v", name, err)
		}

		if insertDummyData {
			sql = fmt.Sprintf(`
			INSERT INTO %s (name, toppings) VALUES
				('Margherita', 'Tomato, Mozzarella, Basil'),
				('Pepperoni', 'Tomato, Mozzarella, Pepperoni'),
				('BBQ Chicken', 'BBQ Sauce, Chicken, Onion, Cheese')
			ON CONFLICT DO NOTHING;
			`, name)

			_, err = db.Exec(sql)
			if err != nil {
				log.Fatalf("Failed to insert dummy data into %s table: %v", name, err)
			}
		}

		fmt.Printf("%s database set up successfully.\n", name)
	} else {
		fmt.Printf("Skipping... %s database already exists.\n", name)
	}
}

func main() {
	initDB("pizza", true)
	initDB("delivery", false)
}
