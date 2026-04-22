//Package database operations
package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	"log"
)

type Value struct {
	ID int
	Symbol string
	Value float64
	Date string
}

var db *sql.DB

func InitDB() {
	var err error
	db, err = sql.Open("sqlite3", "./db/values.db")
	if err != nil {
		log.Fatal(err)
	}
}

func GetBySymbol(str string) ([]Value, error) {
	var res []Value

	query := `SELECT * FROM values WHERE symbol = ?`
	rows, err := db.Query(query, str)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var val Value
		if err := rows.Scan(&val.Symbol, &val.Value, &val.Date); err != nil {
			return res, err
		}
		res = append(res, val)
	}

	if err := rows.Err(); err != nil {
		return res, err
	}
	
	return res, nil
}

func GetByDate(date string) ([]Value, error) {
	var res []Value

	query := `SELECT * FROM values_table WHERE date BETWEEN "1970-01-01" AND ? ORDER BY ABS(strftime("%s", date) - strftime("%s", ?)) ASC`
	rows, err := db.Query(query, date, date)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var val Value
		if err := rows.Scan(&val.ID, &val.Symbol, &val.Value, &val.Date); err != nil {
			return res, err
		}
		res = append(res, val)
	}

	if err := rows.Err(); err != nil {
		return res, err
	}
	
	return res, nil
}

func UpToDate(date string) bool {
	row := db.QueryRow("SELECT * FROM last_update")
	var last string
	err := row.Scan(&last)
	if err != nil {
		log.Println(err)
		return true
	}

	if last != date {
		_, err := db.Exec("UPDATE last_update SET date = ?", date)
		log.Println(err)
		return false
	} else {
		return true
	}
}

func AddRow(val Value) error {
	_, err := db.Exec("INSERT INTO values_table (symbol, value, date) VALUES (?, ?, ?)", val.Symbol, val.Value, val.Date)
	if err != nil {
		return err
	}
	return nil
}
