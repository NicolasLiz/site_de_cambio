package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	
	if convApiKey, ok := os.LookupEnv("CONV_API_KEY"); !ok {
		panic("please set CONV_API_KEY in .env file to the key of currencyapi")
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
	fmt.Println(c.FormValue("ammount"))
	fmt.Println(c.FormValue("from"))
	fmt.Println(c.FormValue("to"))
	return c.String(http.StatusOK, c.FormValue("ammount"))
}
