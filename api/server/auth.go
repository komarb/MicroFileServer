package server

import (
	"MicroFileServer/logging"
	"MicroFileServer/models"
	"github.com/auth0-community/go-auth0"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"net/http"
)
var validator *auth0.JWTValidator
var Claims	models.Claims

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := auth0.NewJWKClient(auth0.JWKClientOptions{URI: cfg.Auth.KeyURL}, nil)
		audience := cfg.Auth.Audience
		configuration := auth0.NewConfiguration(client, []string{audience}, cfg.Auth.Issuer, jose.RS256)
		validator = auth0.NewValidator(configuration, nil)

		_, err := validator.ValidateRequest(r)
		if err != nil {
			log.WithFields(log.Fields{
				"requiredAlgorithm" : "RS256",
				"error" : err,
			}).Warning("Token is not valid!")

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Token is not valid!\nError: "))
			w.Write([]byte(err.Error()))
			return
		}

		Claims = models.Claims{}
		err = getClaims(r)
		if err != nil {
			log.WithFields(log.Fields{
				"requiredClaims" : "iss, aud, sub, itlab",
				"error" : err,
			}).Warning("Invalid claims!")

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid claims!"))
			w.Write([]byte(err.Error()))
			return
		}

		if !checkScope(cfg.Auth.Scope) {
			log.WithFields(log.Fields{
				"requiredScope" : cfg.Auth.Scope,
				"error" : err,
			}).Warning("Invalid scope!")

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid scope"))
			return
		}

		if !isUser() {
			log.WithFields(log.Fields{
				"Claims.ITLab" : Claims.ITLab,
				"function" : "authMiddleware",
			}).Warning("Wrong itlab claim!")
			w.WriteHeader(403)
			w.Write([]byte("Wrong itlab claim!"))
			return
		}

		sw := logging.NewStatusWriter(w)
		next.ServeHTTP(sw, r)
		logging.LogHandler(sw, r)
	})
}

func testAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := []byte("test")
		secretProvider := auth0.NewKeyProvider(secret)
		configuration := auth0.NewConfigurationTrustProvider(secretProvider, nil, "")
		validator = auth0.NewValidator(configuration, nil)
		_, err := validator.ValidateRequest(r)

		if err != nil {
			log.WithFields(log.Fields{
				"requiredAlgorithm" : "HS256",
				"error" : err,
			}).Warning("Token is not valid!")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Token is not valid\nError: "))
			w.Write([]byte(err.Error()))
			return
		}
		getClaims(r)
		sw := logging.NewStatusWriter(w)
		next.ServeHTTP(sw, r)
		logging.LogHandler(sw, r)
	})
}

func checkScope(scope string) bool {
	for _, elem := range Claims.Scope {
		if elem == scope {
			return true
		}
	}
	return false
}

func getClaims(r *http.Request) error {
	token, err := validator.ValidateRequest(r)
	if err != nil {
		return err
	}
	err = validator.Claims(r, token, &Claims)
	return nil
}

func isUser() bool {
	for _, elem := range Claims.ITLab {
		if elem == "user" {
			return true
		}
	}
	return false
}

func isAdmin() bool {
	for _, elem := range Claims.ITLab {
		if elem == "reports.admin" {
			return true
		}
	}
	return false
}