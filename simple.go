package main

// Open url in browser:
// http://localhost:14000/app

import (
	"flag"
	"github.com/RangelReale/osin"
	"github.com/austo/oauth-sample/handlers"
	"net/http"
)

var (
	clientPort     = flag.String("p", "7396", "DK5 client port")
	clientHostname = flag.String("c", "", "DK5 client hostname")
)

func main() {
	flag.Parse()
	cfg := osin.NewServerConfig()
	cfg.AllowGetAccessRequest = true
	cfg.AllowClientSecretInParams = true
	cfg.AccessExpiration = 60

	cfg.AllowedAccessTypes = osin.AllowedAccessType{
		osin.AUTHORIZATION_CODE,
		osin.REFRESH_TOKEN}

	server := osin.NewServer(cfg, NewStorage(*clientHostname, *clientPort))

	authHandler := handlers.NewAuthHandler(server)

	// Application home endpoint
	http.HandleFunc("/app", handleApp)

	// Login endpoint
	http.HandleFunc("/login", authHandler.HandleLogin)

	// Authorization code endpoint
	http.HandleFunc("/authorize", authHandler.HandleAuthorization)

	// Access token endpoint
	http.HandleFunc("/token", authHandler.HandleToken)

	// Information endpoint
	http.HandleFunc("/info", authHandler.HandleInfo)

	http.HandleFunc("/secret", authHandler.HandleSecret)

	// Check access
	http.HandleFunc("/check", authHandler.CheckAccess)

	// Application destination - CODE
	http.HandleFunc("/appauth/code", handleCode)

	http.ListenAndServe("0.0.0.0:14000", nil)
}
