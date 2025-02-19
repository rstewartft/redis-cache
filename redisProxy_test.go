package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"redis-proxy/controllers"
	"redis-proxy/handlers"
	"testing"
	"time"
)

type redisTestSuite struct {
	suite.Suite
	redisClient *redis.Client
	echo        *echo.Echo
}

func (s *redisTestSuite) SetupSuite() {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	fmt.Sprintf("host: %s, port: %s", host, port)
	s.redisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})
	s.echo = echo.New()
}

func (s *redisTestSuite) TearDownSuite() {
	s.redisClient.Close()
}

func TestLocationsSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests")
	}
	suite.Run(t, new(redisTestSuite))
}

// TestLRUCacheOnly tests the case where the key is in the LRU cache but not in redis
func (s *redisTestSuite) TestLRUCacheOnly() {
	testCases := []struct {
		name             string
		keyToInsert      string
		valueToInsert    string
		keyToGet         string
		expectedHttpCode int
	}{
		{
			name:             "success",
			keyToInsert:      "key",
			valueToInsert:    "value",
			keyToGet:         "key",
			expectedHttpCode: http.StatusOK,
		},
		{
			name:             "invalid key",
			keyToInsert:      "key2",
			valueToInsert:    "value",
			keyToGet:         "invalid",
			expectedHttpCode: http.StatusInternalServerError,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			cache := expirable.NewLRU[string, string](10, nil, time.Millisecond*time.Duration(1000000))
			cache.Add(tc.keyToInsert, tc.valueToInsert)
			controller := controllers.NewRedisProxyController(cache, s.redisClient)
			handler := handlers.NewRedisProxy(controller)

			rec := httptest.NewRecorder()
			q := make(url.Values)
			q.Set("key", tc.keyToGet)
			req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
			c := s.echo.NewContext(req, rec)
			assert.NoError(s.T(), handler.GetHTTP(c))

			assert.Equal(s.T(), tc.expectedHttpCode, rec.Code)
			if tc.expectedHttpCode == http.StatusOK {
				assert.Equal(s.T(), fmt.Sprintf("\"%s\"\n", tc.valueToInsert), rec.Body.String())
			}
		})
	}
}

// TestInRedisButNotLRUCache tests the case where the key is in redis but not in the LRU cache
func (s *redisTestSuite) TestInRedisButNotLRUCache() {
	testCases := []struct {
		name             string
		keyToInsert      string
		valueToInsert    string
		keyToGet         string
		expectedHttpCode int
	}{
		{
			name:             "success",
			keyToInsert:      "key",
			valueToInsert:    "value",
			keyToGet:         "key",
			expectedHttpCode: http.StatusOK,
		},
		{
			name:             "invalid key",
			keyToInsert:      "key2",
			valueToInsert:    "value",
			keyToGet:         "invalid",
			expectedHttpCode: http.StatusInternalServerError,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			cache := expirable.NewLRU[string, string](10, nil, time.Millisecond*time.Duration(1000000))
			s.redisClient.Set(context.Background(), tc.keyToInsert, tc.valueToInsert, 0)
			controller := controllers.NewRedisProxyController(cache, s.redisClient)
			handler := handlers.NewRedisProxy(controller)

			rec := httptest.NewRecorder()
			q := make(url.Values)
			q.Set("key", tc.keyToGet)
			req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
			c := s.echo.NewContext(req, rec)
			assert.NoError(s.T(), handler.GetHTTP(c))

			assert.Equal(s.T(), tc.expectedHttpCode, rec.Code)
			if tc.expectedHttpCode == http.StatusOK {
				assert.Equal(s.T(), fmt.Sprintf("\"%s\"\n", tc.valueToInsert), rec.Body.String())
				// if not in LRU cache, should get added after this get call.
				val, exists := cache.Get(tc.keyToGet)
				assert.True(s.T(), exists)
				assert.Equal(s.T(), tc.valueToInsert, val)
			}
		})
	}
}

// TestLRUCacheParams tests configurable parameters for the LRU cache. Assumes values are in redis but not local cache
func (s *redisTestSuite) TestLRUCacheParams() {
	testCases := []struct {
		name             string
		keyToInsert      string
		valueToInsert    string
		keyToGet         string
		cacheSize        int
		globalExpiry     int
		expectedHttpCode int
	}{
		{
			name:             "eviction upon cache size limit",
			keyToInsert:      "key",
			valueToInsert:    "value",
			keyToGet:         "key",
			cacheSize:        1,
			globalExpiry:     0,
			expectedHttpCode: http.StatusOK,
		},
		{
			name:             "eviction upon global expiry",
			keyToInsert:      "key2",
			valueToInsert:    "value",
			keyToGet:         "key2",
			cacheSize:        0,
			globalExpiry:     100,
			expectedHttpCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			cache := expirable.NewLRU[string, string](tc.cacheSize, nil, time.Millisecond*time.Duration(tc.globalExpiry))
			// add key to both redis and local cache.  local cache version should expire based on test params.
			cache.Add(tc.keyToInsert, tc.valueToInsert)
			s.redisClient.Set(context.Background(), tc.keyToInsert, tc.valueToInsert, 0)
			controller := controllers.NewRedisProxyController(cache, s.redisClient)
			handler := handlers.NewRedisProxy(controller)

			if tc.globalExpiry > 0 {
				time.Sleep(time.Millisecond * time.Duration(tc.globalExpiry))
			}
			if tc.cacheSize == 1 {
				cache.Add("blah", "blahVal")
			}
			// before the get call, the key should not be in the cache
			_, exists := cache.Get(tc.keyToGet)
			assert.False(s.T(), exists)

			rec := httptest.NewRecorder()
			q := make(url.Values)
			q.Set("key", tc.keyToGet)
			req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
			c := s.echo.NewContext(req, rec)
			assert.NoError(s.T(), handler.GetHTTP(c))
			// after the get call, the key should be in the cache
			val, exists := cache.Get(tc.keyToGet)
			assert.True(s.T(), exists)
			assert.Equal(s.T(), tc.valueToInsert, val)

			if tc.expectedHttpCode == http.StatusOK {
				assert.Equal(s.T(), fmt.Sprintf("\"%s\"\n", tc.valueToInsert), rec.Body.String())
				// if not in LRU cache, should get added after this get call.
				val, exists = cache.Get(tc.keyToGet)
				assert.True(s.T(), exists)
				assert.Equal(s.T(), tc.valueToInsert, val)
			}
		})
	}
}

// TestRESP tests getting values from the cache using the RESP protocol
func (s *redisTestSuite) TestRESP() {
	testCases := []struct {
		name          string
		keyToInsert   string
		valueToInsert string
		keyToGet      string
		message       string
		expectedResp  string
	}{
		{
			name:          "success",
			keyToInsert:   "testkey",
			valueToInsert: "testvalue",
			keyToGet:      "testkey",
			message:       "*2\r\n$3\r\nGET\r\n$7\r\ntestkey\r\n",
			expectedResp:  "$9\r\ntestvalue\r\n",
		},
		{
			name:          "invalid key",
			keyToInsert:   "key2",
			valueToInsert: "value",
			keyToGet:      "invalid",
			message:       "*2\r\n$3\r\nGET\r\n$5\r\nhello\r\n",
			expectedResp:  "-Error key not found\r\n",
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			cache := expirable.NewLRU[string, string](10, nil, time.Millisecond*time.Duration(1000000))
			cache.Add(tc.keyToInsert, tc.valueToInsert)
			controller := controllers.NewRedisProxyController(cache, s.redisClient)
			handler := handlers.NewRedisProxy(controller)

			go func() {
				conn, err := net.Dial("tcp", "localhost:6381")
				if err != nil {
					t.Fatal(err)
					return
				}
				defer conn.Close()

				if _, err := fmt.Fprintf(conn, tc.message); err != nil {
					t.Fatal(err)
				}
			}()

			l, err := net.Listen("tcp", "localhost:6381")
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()

			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				response := handler.GetRESP(conn)
				assert.Equal(t, tc.expectedResp, response)
				return
			}
		})
	}
}
