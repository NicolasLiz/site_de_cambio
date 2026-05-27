//Package database operations
package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	"log"
	"os"
)

type Value struct {
	ID int
	Symbol string
	Value float64
	Date string
}

var db *sql.DB

func InitDB() {
    // Garante que a pasta /app/db exista no Render
    if _, err := os.Stat("./db"); os.IsNotExist(err) {
        os.Mkdir("./db", os.ModePerm)
    }

    var err error
    db, err = sql.Open("sqlite3", "./db/values.db")
    if err != nil {
        log.Fatal(err)
    }
}

func GetHistorical(str string) ([]Value, error) {
	var res []Value

	query := `SELECT * FROM values_table WHERE symbol = ? ORDER BY strftime("%s", date)`
	rows, err := db.Query(query, str)
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

func GetLatest(symbol string) (Value, error) {
	query := `SELECT * FROM values_table WHERE symbol = ? ORDER BY strftime("%s", date) DESC LIMIT 1`
	row := db.QueryRow(query, symbol)

	var val Value
	if err := row.Scan(&val.ID, &val.Symbol, &val.Value, &val.Date); err != nil {
		return Value{}, err
	}
	
	return val, nil
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

// Trend represents the price movement of a currency compared to its previous value
type Trend struct {
	Symbol        string  `json:"symbol"`
	CurrentValue  float64 `json:"current_value"`
	PreviousValue float64 `json:"previous_value"`
	PercentChange float64 `json:"percent_change"` // e.g., 2.5 for +2.5%, -1.2 for -1.2%
}

// GetMarketTrends calculates the percentage change between the latest and the 
// second-to-latest entries for all unique currencies.
func GetMarketTrends() ([]Trend, error) {
	var trends []Trend

	// This query ranks entries per symbol by date descending, grabs the top 2,
	// and aggregates them so we can compare the latest value to the previous one.
	query := `
		WITH RankedValues AS (
			SELECT 
				symbol, 
				value,
				ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY strftime('%s', date) DESC) as rn
			FROM values_table
		)
		SELECT 
			v1.symbol,
			v1.value AS current_value,
			COALESCE(v2.value, v1.value) AS previous_value,
			CASE 
				WHEN v2.value IS NOT NULL AND v2.value != 0 
				THEN ((v1.value - v2.value) / v2.value) * 100
				ELSE 0.0 
			END AS percent_change
		FROM RankedValues v1
		LEFT JOIN RankedValues v2 ON v1.symbol = v2.symbol AND v2.rn = 2
		WHERE v1.rn = 1;
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Println("Error fetching market trends:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t Trend
		err := rows.Scan(&t.Symbol, &t.CurrentValue, &t.PreviousValue, &t.PercentChange)
		if err != nil {
			return nil, err
		}
		trends = append(trends, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trends, nil
}

// GetByDate pulls all entries matching a specific date string
func GetByDate(date string) ([]Value, error) {
	var res []Value
	query := `SELECT id, symbol, value, date FROM values_table WHERE date = ?`
	rows, err := db.Query(query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var val Value
		if err := rows.Scan(&val.ID, &val.Symbol, &val.Value, &val.Date); err != nil {
			return nil, err
		}
		res = append(res, val)
	}
	return res, nil
}
