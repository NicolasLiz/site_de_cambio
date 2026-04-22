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

	//getting previous 10 months values
	if err := getHistoricalValues("2025-12-01"); err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.Use(middleware.RequestLogger())

	e.Static("/static", "static")
	e.File("/", "static/html/index.html")
	e.File("/script.js", "static/js/script.js")

	e.POST("/convert", convertCoins)
	e.POST("/graph-data", graphData)

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
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

	fmt.Println(body)

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
