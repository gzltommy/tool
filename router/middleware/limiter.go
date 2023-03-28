package middleware

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"tool-attendance/utils"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"golang.org/x/time/rate"
)

// r:代表每秒可以向Token桶中产生多少token， b：代表Token桶的容量大小
var limiter = NewIPRateLimiter(1, 1)

var loginLimiter = NewIPRateLimiter(1, 2)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

type userAddressLimit struct {
	Address string `json:"address"`
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	return i
}

// AddIP creates a new rate limiter and adds it to the ips map,
// using the IP address as the key
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)

	i.ips[ip] = limiter

	return limiter
}

// GetLimiter returns the rate limiter for the provided IP address if it exists.
// Otherwise calls AddIP to add IP address to the map
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]

	if !exists {
		i.mu.Unlock()
		return i.AddIP(ip)
	}

	i.mu.Unlock()

	return limiter
}

func LimitMiddleware(c *gin.Context) {
	var body_ []byte
	if c.Request.Body != nil {
		body_, _ = ioutil.ReadAll(c.Request.Body)
	}

	if c.Request.Method == "OPTIONS" {
		c.Next()
		return
	}
	ip := utils.GetClientIP(c.Request)

	var userAddress string
	var uAddress userAddressLimit
	if c.ContentType() == "application/json" {
		_ = c.ShouldBindBodyWith(&uAddress, binding.JSON)
	}
	if uAddress.Address == "" {
		userAddress = c.PostForm("address")
		if userAddress == "" {
			userAddress = c.Query("address")
		}
	} else {
		userAddress = uAddress.Address
	}
	limiter := limiter.GetLimiter(ip + userAddress)
	if !limiter.Allow() {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"result": false,
			"error":  "Too Many Requests",
		})
		return
	}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body_))
	c.Next()
}

func LoginLimitMiddleware(c *gin.Context) {
	l := loginLimiter.GetLimiter(c.ClientIP())
	if !l.Allow() {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"result": false,
			"error":  "Too Many Requests",
		})
		return
	}
	c.Next()
}
