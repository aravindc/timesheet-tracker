package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// getWhitelistedIPs reads allowed IPs from WHITELIST_IPS env var.
func getWhitelistedIPs() []string {
	ipsStr := os.Getenv("WHITELIST_IPS")
	if ipsStr == "" {
		log.Println("Warning: No WHITELIST_IPS configured, defaulting to localhost only")
		return []string{"127.0.0.1", "::1"}
	}

	ips := strings.Split(ipsStr, ",")
	var cleanIPs []string
	for _, ip := range ips {
		cleanIP := strings.TrimSpace(ip)
		if cleanIP != "" {
			cleanIPs = append(cleanIPs, cleanIP)
		}
	}

	log.Printf("Loaded whitelisted IPs: %v", cleanIPs)
	return cleanIPs
}

func getWhitelistedCIDRs() []string {
	cidrsStr := os.Getenv("WHITELIST_CIDRS")
	if cidrsStr == "" {
		return []string{}
	}

	cidrs := strings.Split(cidrsStr, ",")
	var cleanCIDRs []string
	for _, cidr := range cidrs {
		cleanCIDR := strings.TrimSpace(cidr)
		if cleanCIDR != "" {
			cleanCIDRs = append(cleanCIDRs, cleanCIDR)
		}
	}

	log.Printf("Loaded whitelisted CIDRs: %v", cleanCIDRs)
	return cleanCIDRs
}

// getTrustedProxies reads trusted proxy IPs from TRUSTED_PROXIES env var.
func getTrustedProxies() []string {
	proxiesStr := os.Getenv("TRUSTED_PROXIES")
	if proxiesStr == "" {
		return []string{"127.0.0.1", "::1"}
	}

	proxies := strings.Split(proxiesStr, ",")
	var cleanProxies []string
	for _, proxy := range proxies {
		cleanProxy := strings.TrimSpace(proxy)
		if cleanProxy != "" {
			cleanProxies = append(cleanProxies, cleanProxy)
		}
	}

	return cleanProxies
}

// IPWhitelistMiddleware blocks requests from IPs not in allowedIPs.
func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		allowed := false
		for _, ip := range allowedIPs {
			if clientIP == ip {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: IP not whitelisted", "ClientIP": clientIP, "AllowedIPs": allowedIPs})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRangeWhitelistMiddleware blocks requests from IPs not within allowedCIDRs.
func IPRangeWhitelistMiddleware(allowedCIDRs []string) gin.HandlerFunc {
	var ipNets []*net.IPNet
	for _, cidr := range allowedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			ipNets = append(ipNets, ipNet)
		}
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		ip := net.ParseIP(clientIP)

		if ip == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid IP address"})
			c.Abort()
			return
		}

		allowed := false
		for _, ipNet := range ipNets {
			if ipNet.Contains(ip) {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: IP not in allowed range"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// authMiddleware validates the JWT Bearer token and sets user_id/username in context.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// LoggerWithIP is a gin logger that includes client IP in the output.
func LoggerWithIP() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s | %3d | %13v | %15s | %-7s %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// oauthCallbackPaths carry one-time authorization codes and state in the query
// string, so the default request logger must not record them.
var oauthCallbackPaths = []string{
	"/api/auth/google/callback",
	"/api/auth/facebook/callback",
}

// zapMiddlewares returns the ginzap logging and recovery middleware pair.
func zapMiddlewares(s *Server) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		ginzap.GinzapWithConfig(s.logger, &ginzap.Config{
			TimeFormat: time.RFC3339,
			UTC:        true,
			SkipPaths:  oauthCallbackPaths,
		}),
		ginzap.RecoveryWithZap(s.logger, true),
	}
}

// requestLoggerWithoutQuery logs a request like ginzap but omits the query string.
func (s *Server) requestLoggerWithoutQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		s.logger.Info(c.Request.URL.Path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}
