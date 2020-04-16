package server

import (
	"MicroFileServer/logging"
	"MicroFileServer/models"
	"errors"
	"github.com/auth0-community/go-auth0"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"net/http"
)
var validator *auth0.JWTValidator
var Claims	models.Claims

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := logging.NewStatusWriter(w)
		next.ServeHTTP(sw, r)
		logging.LogHandler(sw, r)
	})
}

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
		Claims.ITLab = nil
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
		Claims.ITLab = nil
		getClaims(r)

		log.Info(Claims.ITLab)
		log.Info(Claims.Sub)

		sw := logging.NewStatusWriter(w)
		next.ServeHTTP(sw, r)
		logging.LogHandler(sw, r)
	})
}



func getClaims(r *http.Request) error {
	token, err := validator.ValidateRequest(r)
	if err != nil {
		return err
	}
	err = validator.Claims(r, token, &Claims)

	switch Claims.ITLabInterface.(type) {
	case string:
		claimString := Claims.ITLabInterface.(string)
		Claims.ITLab = []string{claimString}
	case []interface{}:
		claimInterface, ok := Claims.ITLabInterface.([]interface{})
		claimString := make([]string, len(claimInterface))
		for i, v := range claimInterface {
			claimString[i], ok = v.(string)
			if !ok { return errors.New("itLab claim is invalid") }
		}
		if ok { Claims.ITLab = claimString }
	default:
		return errors.New("itLab claim is invalid")
	}
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
		if elem == "mfs.admin" {
			return true
		}
	}
	return false
}