package jwt

import (
	"context"
	"sync"
	"time"

	jwtv2 "github.com/gogf/gf-jwt/v2"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

type JwtConfig struct {
	Realm      string
	Key        string
	Timeout    int
	MaxRefresh int
}

var authService *jwtv2.GfJWTMiddleware

var once sync.Once

func NewJwt(ctx context.Context) *jwtv2.GfJWTMiddleware {
	once.Do(func() {
		// 配置文件
		config := &JwtConfig{}
		err := g.Cfg().MustGet(ctx, "jwt").Scan(config)
		if err != nil {
			gerror.Wrap(err, "jwt 配置错误")
			return
		}

		auth := jwtv2.New(&jwtv2.GfJWTMiddleware{
			Realm:           config.Realm,
			Key:             []byte(config.Key),
			Timeout:         time.Minute * time.Duration(config.Timeout),
			MaxRefresh:      time.Minute * time.Duration(config.MaxRefresh),
			IdentityKey:     "id",
			TokenLookup:     "header: Authorization, query: token, cookie: jwt",
			TokenHeadName:   "Bearer",
			TimeFunc:        time.Now,
			Authenticator:   Authenticator,
			Unauthorized:    Unauthorized,
			PayloadFunc:     PayloadFunc,
			IdentityHandler: IdentityHandler,
		})
		authService = auth
	})
	return authService
}

func PayloadFunc(data interface{}) jwtv2.MapClaims {
	claims := jwtv2.MapClaims{}
	params := data.(map[string]interface{})
	if len(params) > 0 {
		for k, v := range params {
			claims[k] = v
		}
	}
	return claims
}

func IdentityHandler(ctx context.Context) interface{} {
	claims := jwtv2.ExtractClaims(ctx)
	return claims[authService.IdentityKey]
}

func GetUserID(ctx context.Context) uint64 {
	if authService == nil {
		return 0
	}

	claims, _, err := authService.GetClaimsFromJWT(ctx)
	if err != nil {
		return 0
	}

	return gconv.Uint64(claims["id"])
}

func Unauthorized(ctx context.Context, code int, message string) {
	r := g.RequestFromCtx(ctx)
	r.Response.WriteHeader(401)
	r.Response.WriteJson(g.Map{
		"message": message,
	})
}

// Authenticator is used to validate login parameters.
// It must return user data as user identifier, it will be stored in Claim Array.
// if your identityKey is 'id', your user data must have 'id'
// Check error (e) to determine the appropriate error message.
func Authenticator(ctx context.Context) (user interface{}, err error) {
	return user, nil
}
