package filter_impl

import (
	"errors"
	"os"
	"strings"

	"github.com/KKKKjl/tinykit/internal/context"
	"github.com/KKKKjl/tinykit/internal/filter"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/viper"
)

var (
	Secret           []byte
	SigningAlgorithm string

	AuthKey             = "Authorization"
	TokenNotFoundErr    = errors.New("Required authorization token not found.")
	TokenStructErr      = errors.New("Token struct error.")
	InvalidSignatureErr = errors.New("Invalid signing algorithm.")
)

type User map[string]interface{}

func init() {
	secret, ok := os.LookupEnv("TINYKIT_JWT_SECRET")
	if !ok {
		//panic("Jwt secret not found in env.")
	}

	SigningAlgorithm = viper.GetString("SIGNING_AlGORITHM")
	Secret = []byte(secret)
}

func JwtFilter() filter.HandleFilter {
	return func(ctx context.HttpContext, next filter.Next) {
		tokenStr, err := getTokenFromHeader(ctx)
		if err != nil {
			ctx.Error(err)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if jwt.GetSigningMethod(SigningAlgorithm) != token.Method {
				return nil, InvalidSignatureErr
			}

			return Secret, nil
		})
		if err != nil {
			if errors.Is(err, jwt.ErrTokenMalformed) {
				ctx.AbortWithMsg("Invalid token.")
			} else if errors.Is(err, jwt.ErrTokenExpired) {
				ctx.AbortWithMsg("Token expired.")
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				ctx.AbortWithMsg("Token not active yet.")
			} else {
				ctx.Error(err)
			}

			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !(ok && token.Valid) {
			ctx.Error(TokenStructErr)
			return
		}

		id := claims["id"].(string)
		ctx.SetValue(User{}, id)

		next(ctx)
	}
}

func getTokenFromHeader(ctx context.HttpContext) (string, error) {
	authorization := ctx.Request.Header.Get(AuthKey)
	if authorization == "" {
		return "", TokenNotFoundErr
	}

	parts := strings.Fields(authorization)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", TokenStructErr
	}

	return parts[1], nil
}
