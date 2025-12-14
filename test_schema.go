package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schema := `CREATE TABLE test (
		id INTEGER PRIMARY KEY,
		"exists" BOOLEAN,
		name TEXT
	)`

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("PRAGMA table_info(test)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Table columns:")
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ, dflt_value sql.NullString
		err = rows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Column: %s, Type: %s\n", name.String, typ.String)
	}

	// Test INSERT
	_, err = db.Exec(`INSERT INTO test ("exists", name) VALUES (?, ?)`, true, "test")
	if err != nil {
		log.Fatal("INSERT failed:", err)
	}

	fmt.Println("INSERT succeeded")
}
