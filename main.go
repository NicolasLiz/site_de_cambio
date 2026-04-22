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

	e.POST("/convert", convertCoins)

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

func convertCoins(c *echo.Context) error {
	log.Println(c.FormValue("ammount"), c.FormValue("from"), c.FormValue("to"))
	return c.String(http.StatusOK, c.FormValue("ammount"))
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

	fmt.Printf("%v", latest)

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
