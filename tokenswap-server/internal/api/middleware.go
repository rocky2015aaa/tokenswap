package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rocky2015aaa/tokenswap-server/internal/config"
	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/database"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	BearerPrefix = "Bearer"
)

var (
	userRoles   = map[string]struct{}{} // To use free / paid account
	publicPaths = map[string]struct{}{
		"/api/v1/health":        {},
		"/api/v1/user/register": {},
		"/api/v1/token/refresh": {},
		"/api/v1/token/renew":   {},
	}
)

// var userRole string

func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		// Now for testing, no cors filter but in the future, will add preventing ones.
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	})
}

func UserRoleHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}

func UserAuthentication() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if _, ok := publicPaths[ctx.FullPath()]; !ok {
			// Parse http request header
			authHeader := ctx.GetHeader("Authorization")
			if len(authHeader) == 0 {
				err := fmt.Errorf("there is no authorization token")
				log.Error(err)
				handleResponse(ctx, http.StatusUnauthorized, err.Error())
				return
			}
			if !strings.HasPrefix(authHeader, BearerPrefix+" ") {
				err := fmt.Errorf("invalid authorization format")
				log.Error(err)
				handleResponse(ctx, http.StatusUnauthorized, err.Error())
				return
			}
			tokenString := authHeader[len(BearerPrefix)+1:]
			// Parse the jwt token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv(config.EnvStsvrJwtSecret)), nil
			})
			if err != nil {
				log.Error(err)
				handleResponse(ctx, http.StatusUnauthorized, err.Error())
				return
			}
			if token.Valid {
				// Extract claims from the token
				claims, ok := token.Claims.(jwt.MapClaims)
				if !ok {
					err := fmt.Errorf("invalid claims format")
					log.Error(err)
					handleResponse(ctx, http.StatusUnauthorized, err.Error())
					return
				}
				if ctx.FullPath() == "/api/v1/user/" {
					// pass token_expiration_date_time for the user information
					if exp, ok := claims["exp"].(float64); ok {
						ctx.Set("token_expiration_date_time", time.Unix(int64(exp), 0).Format(database.TimeFormat))
					} else {
						err := fmt.Errorf("token does not contain an 'exp' claim")
						log.Error(err)
						handleResponse(ctx, http.StatusUnauthorized, err.Error())
						return
					}
				}
				uuid, ok := claims["username"].(string)
				if !ok {
					err := fmt.Errorf("token does not contain an 'username' claim")
					log.Error(err)
					handleResponse(ctx, http.StatusUnauthorized, err.Error())
					return
				}
				ctx.Set("uuid", uuid)
			} else {
				err := fmt.Errorf("jwt token is not valid")
				log.Error(err)
				handleResponse(ctx, http.StatusUnauthorized, err.Error())
				return
			}
		}

		ctx.Next()
	}
}

func HttpMethodChecker(router *gin.Engine) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		method := ctx.Request.Method
		route := ctx.Request.URL.Path
		routes := router.Routes()

		for _, r := range routes {
			if r.Path == route {
				if r.Method != method {
					handleResponse(ctx, http.StatusMethodNotAllowed, "Method Not Allowed")
					return
				}
				break
			}
		}

		ctx.Next()
	}
}

func handleResponse(ctx *gin.Context, statusCode int, errMsg string) {
	ctx.AbortWithStatusJSON(statusCode, gin.H{
		"success":     true,
		"data":        "",
		"error":       errMsg,
		"description": "",
	})
}
