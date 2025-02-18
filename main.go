package main

import (
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"net"
	"os"
	"redis-proxy/controllers"
	"redis-proxy/handlers"
	"strconv"
	"time"
)

func main() {
	e := echo.New()

	// get configuration from environment variables
	var (
		keySizeInt      int
		globalExpiryInt int
		err             error

		globalExpiry = os.Getenv("GLOBAL_EXPIRY_MS")
		keySize      = os.Getenv("KEY_SIZE")
		redisPort    = os.Getenv("REDIS_PORT")
		redisHost    = os.Getenv("REDIS_HOST")
		httpAddress  = os.Getenv("HTTP_ADDRESS")
		respAddress  = os.Getenv("RESP_ADDRESS")
	)
	if keySize != "" {
		keySizeInt, err = strconv.Atoi(keySize)
		if err != nil {
			e.Logger.Fatal(err)
		}
	}
	if globalExpiry != "" {
		globalExpiryInt, err = strconv.Atoi(globalExpiry)
		if err != nil {
			e.Logger.Fatal(err)
		}
	}

	// initialize the cache and redis client
	cache := expirable.NewLRU[string, string](keySizeInt, nil, time.Millisecond*time.Duration(globalExpiryInt))
	rdb := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
	})
	controller := controllers.NewRedisProxyController(cache, rdb)
	handler := handlers.NewRedisProxy(controller)

	// listen on tcp port for RESP requests
	l, err := net.Listen("tcp", respAddress)
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer l.Close()
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				e.Logger.Fatal(err)
			}
			go handler.GetRESP(conn)
		}
	}()

	// listen on http port for REST requests
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(20))))
	e.GET("/", handler.GetHTTP)
	if err = e.Start(httpAddress); err != nil {
		e.Logger.Fatal(err)
	}
}
