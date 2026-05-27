package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"io"
	"bytes"
	"encoding/json"
	"strings"
	"strconv"

	"moeda/internal/database"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/joho/godotenv"
)

type LatestValues struct {
	Meta struct {
		LastUpdated string `json:"last_updated_at"`
	} `json:"meta"`
	Data map[string]struct {
		Code string `json:"code"`
		Value float64 `json:"value"`
	} `json:"data"`
}

var convAPIKey string

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	var ok bool
	if convAPIKey, ok = os.LookupEnv("CONV_API_KEY"); !ok {
		panic("please set CONV_API_KEY in .env file to the key of currencyapi")
	}

	database.InitDB()
	y, m, d := time.Now().Date()
	today := fmt.Sprintf("%.2d-%.2d-%.2d", y, m, d)

	if !database.UpToDate(today) {
		err := getCurrentValues()
		if err != nil {
			log.Println(err)
		}
	}

	e := echo.New()
	e.Use(middleware.RequestLogger())

	e.Static("/static", "static")
	e.File("/", "static/html/index.html")
	e.File("/script.js", "static/js/script.js")

	e.POST("/api/v1/convert", convertCoins)
	e.POST("/api/v1/trends", getTrendsHandler)

	// get all values from date
	e.POST("/api/v1/request", requestDate)

	// get history of a coin
	e.POST("/api/v1/history", requestHistory)

	a, _ := database.GetMarketTrends()

	fmt.Printf("%v", a)

	e.POST("/graph-data", graphData)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080" // Porta padrão caso esteja rodando localmente
    }

	if err := e.Start(":" + port); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

func getTrendsHandler(c* echo.Context) error {
	// 1. Fetch the computed trends from the database package
	trends, err := database.GetMarketTrends()
	if err != nil {
		// If something went wrong internally, return a 500 Internal Server Error
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve market trends from the database",
		})
	}

	// 2. If no data exists yet, return an empty array instead of null
	if trends == nil {
		trends = []database.Trend{}
	}

	// 3. Send the structured data back to the frontend
	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    trends,
	})
}

func convertLatest(from, to, ammount string) (float64, error) {
	log.Println(from, to, ammount)
	res := 0.0

	valFrom, err := database.GetLatest(from)
	if err != nil {
		return 0, err
	}

	valTo, err := database.GetLatest(to)
	if err != nil {
		return 0, err
	}

	ammountInt, _ := strconv.ParseFloat(ammount, 64)

	res = (ammountInt / valFrom.Value) * valTo.Value

	return res, nil
}

func convertCoins(c *echo.Context) error {
	res, err := convertLatest(c.FormValue("from"), c.FormValue("to"), c.FormValue("ammount"))
	if err != nil {
		log.Println(err)
	}
	return c.String(http.StatusOK, fmt.Sprintf("%f", res))
}

// func requestDate(c *echo.Context) {
//
// }

func graphData(c *echo.Context) error {
	values, err := database.GetHistorical(c.FormValue("symbol"))
	if err != nil {
		log.Println(err)
	}

	var response bytes.Buffer
	encoder := json.NewEncoder(&response)
	encoder.Encode(&values)
	return c.String(http.StatusOK, response.String())
}

func getCurrentValues() error {
	res, err := APICall("https://api.currencyapi.com/v3/latest")
	if err != nil {
		return err
	}
	
	var body bytes.Buffer
	io.Copy(&body, res.Body)
	res.Body.Close()

	decoder := json.NewDecoder(&body)
	latest := LatestValues{}
	err = decoder.Decode(&latest)
	if err != nil {
		return err
	}

	latest.Meta.LastUpdated = strings.Split(latest.Meta.LastUpdated, "T")[0]

	for k, v := range latest.Data{
		value := database.Value{Symbol:k, Value:v.Value, Date:latest.Meta.LastUpdated}
		err := database.AddRow(value)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func requestHistory(c *echo.Context) error {
	// Read value directly from the submitted HTML form
	symbolStr := strings.ToUpper(strings.TrimSpace(c.FormValue("symbol")))

	if symbolStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing 'symbol' form value",
		})
	}

	// Use your existing package function to query local history
	history, err := database.GetHistorical(symbolStr)
	if err != nil {
		log.Println("Database Error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal database error",
		})
	}

	// If the database has absolutely no records for this asset, baseline it from the API
	if len(history) == 0 {
		log.Printf("No history for %s. Fetching contemporary snapshot...", symbolStr)
		
		today := time.Now().Format("2006-01-02")
		if err := getHistoricalValues(today); err != nil {
			log.Println("API Fallback Error:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Asset not found and fallback sync failed",
			})
		}

		// Pull records again now that the table has records
		history, err = database.GetHistorical(symbolStr)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to load database entries post-sync",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"symbol":  symbolStr,
		"data":    history,
	})
}

func requestDate(c *echo.Context) error {
	// Read value directly from the submitted HTML form
	dateStr := c.FormValue("date")

	if dateStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing 'date' form value",
		})
	}

	// Check if data exists for this date via your UpToDate logic
	isUpToDate := database.UpToDate(dateStr)

	if !isUpToDate {
		log.Printf("Data for %s missing. Triggering API download...", dateStr)
		// Call your existing function to fetch from external API and write to DB
		if err := getHistoricalValues(dateStr); err != nil {
			log.Println("API Error:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to retrieve and cache external API data",
			})
		}
	}

	// Retrieve data from your database to output as JSON
	results, err := database.GetByDate(dateStr) // Uses the database helper mentioned previously
	if err != nil {
		log.Println("Database Error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal database retrieval failed",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"date":    dateStr,
		"data":    results,
	})
}

func getHistoricalValues(date string) error {
	res, err := APICall(fmt.Sprintf("https://api.currencyapi.com/v3/historical?date=%s", date))
	if err != nil {
		return err
	}
	
	var body bytes.Buffer
	io.Copy(&body, res.Body)
	res.Body.Close()

	decoder := json.NewDecoder(&body)
	latest := LatestValues{}
	err = decoder.Decode(&latest)
	if err != nil {
		return err
	}

	latest.Meta.LastUpdated = strings.Split(latest.Meta.LastUpdated, "T")[0]

	for k, v := range latest.Data{
		value := database.Value{Symbol:k, Value:v.Value, Date:latest.Meta.LastUpdated}
		err := database.AddRow(value)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func APICall(endpoint string) (*http.Response, error) {
	client := http.Client{}
	request, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return &http.Response{}, err
	}
	request.Header.Add("apikey", convAPIKey)

	response, err := client.Do(request)
	if err != nil {
		return &http.Response{}, err
	}

	if response.StatusCode != 200 {
		return &http.Response{}, fmt.Errorf("status code not 200: %d", response.StatusCode)
	}

	return response, nil
}
