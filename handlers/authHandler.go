package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/RangelReale/osin"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	body_str = `<html>

<body>
    LOGIN %s (use test/test)
    <br/>
    <form action="/authorize?response_type=%s&client_id=%s&state=%s&redirect_uri=%s&scope=%s" method="POST">
        Login:
        <input type="text" name="login" />
        <br/> Password:
        <input type="password" name="password" />
        <br/>
        <input type="submit" />
    </form>
</body>

</html>`
)

var bearerAuthRe = regexp.MustCompile("^Bearer\\s([\\w\\-]+)$")

type user struct {
	id       int
	name     string
	password string
}

type AuthHandler struct {
	server *osin.Server
	users  map[string]user
}

func NewAuthHandler(s *osin.Server) *AuthHandler {
	handler := new(AuthHandler)
	handler.server = s
	handler.users = make(map[string]user)
	handler.setUsers()
	return handler
}

func (ah *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("HANDLING LOGIN REQUEST: %s\n", r.RequestURI)
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	resp := ah.server.NewResponse()
	defer resp.Close()
	if ar := ah.server.HandleAuthorizeRequest(resp, r); ar != nil {
		fmt.Println(ar)
		w.Write([]byte(getLoginPage(ar)))
		return
	}
	w.WriteHeader(400)
}

func (ah *AuthHandler) HandleAuthorization(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("HANDLING AUTHORIZATION REQUEST: %s\n", r.RequestURI)
	if r.Method != "POST" {
		w.WriteHeader(405)
		w.Write([]byte("only POST allowed"))
		return
	}
	resp := ah.server.NewResponse()
	defer resp.Close()

	if ar := ah.server.HandleAuthorizeRequest(resp, r); ar != nil {
		if !ah.validateLogin(ar, w, r) {
			return
		}
		ar.Authorized = true
		ah.server.FinishAuthorizeRequest(resp, r, ar)
	}
	if resp.IsError && resp.InternalError != nil {
		fmt.Printf("ERROR: %s\n", resp.InternalError)
		w.WriteHeader(500)
		return
	}
	// Library function will redirect if necessary
	osin.OutputJSON(resp, w, r)
}

func (ah *AuthHandler) HandleSecret(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("HANDLING SECRET REQUEST: %s\n", r.RequestURI)
	resp := ah.server.NewResponse()
	defer resp.Close()

	if ah.isAuthorized(r) {
		w.Write([]byte("you have entered the secret area"))
		return
	}
	w.WriteHeader(401)
	w.Write([]byte("unauthorized"))
}

func (ah *AuthHandler) HandleInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("HANDLING INFO REQUEST: %s\n", r.RequestURI)
	resp := ah.server.NewResponse()
	defer resp.Close()

	if ir := ah.server.HandleInfoRequest(resp, r); ir != nil {
		ah.server.FinishInfoRequest(resp, r, ir)
	}
	osin.OutputJSON(resp, w, r)
}

func (ah *AuthHandler) HandleToken(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("HANDLING TOKEN REQUEST: %s\n", r.RequestURI)
	resp := ah.server.NewResponse()
	defer resp.Close()

	if ar := ah.server.HandleAccessRequest(resp, r); ar != nil {
		ar.Authorized = true
		ah.server.FinishAccessRequest(resp, r, ar)
	}
	if resp.IsError && resp.InternalError != nil {
		fmt.Printf("ERROR: %s\n", resp.InternalError)
	}
	osin.OutputJSON(resp, w, r)
}

type checkAccessRequest struct {
	AccessToken string `json:"access_token"`
	RequestUri  string `json:"request_uri"`
}

type checkAccessResponse struct {
	ExpiresIn    int32  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (ah *AuthHandler) CheckAccess(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var checkReq checkAccessRequest
	err := decoder.Decode(&checkReq)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}
	accessData, err := ah.server.Storage.LoadAccess(checkReq.AccessToken)
	if accessData == nil || err != nil {
		fmt.Println(err)
		w.WriteHeader(401)
		return
	}

	if strings.Index(checkReq.RequestUri, accessData.Scope) != 0 {
		w.WriteHeader(401)
		w.Write([]byte("invalid scope"))
	}

	checkRes := checkAccessResponse{
		int32(accessData.CreatedAt.Add(
			time.Duration(accessData.ExpiresIn)*time.Second).Sub(
			ah.server.Now()) / time.Second),
		accessData.RefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	err = enc.Encode(checkRes)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	return
}

func (ah *AuthHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {

	if !ah.isAuthorized(r) {
		sendNotAuthorized(&w, "did not pass step 1")
		return
	}

	header, ok := r.Header["Authorization"]
	if !ok {
		sendNotAuthorized(&w, "did not pass step 2")
		return
	}

	if len(header) < 1 {
		sendNotAuthorized(&w, "did not pass step 3")
		return
	}

	tokenMatch := bearerAuthRe.FindStringSubmatch(header[0])
	if tokenMatch == nil {
		sendNotAuthorized(&w, "did not pass step 4")
		return
	}

	token := tokenMatch[1]

	accessData, err := ah.server.Storage.LoadAccess(token)
	if accessData == nil || err != nil {
		sendNotAuthorized(&w, "did not pass step 5")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err = enc.Encode(accessData.UserData)
	if err != nil {
		fmt.Println(err)
		sendNotAuthorized(&w, "did not pass step 6")
		return
	}
}

func sendNotAuthorized(w *http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "not authorized"
	}
	(*w).Header().Set("Content-Type", "application/json")
	(*w).WriteHeader(401)
	(*w).Write([]byte(fmt.Sprintf("{\"message\": \"%s\"}", msg)))
}

func (ah *AuthHandler) validateLogin(ar *osin.AuthorizeRequest, w http.ResponseWriter, r *http.Request) bool {
	fmt.Println("validating login")
	r.ParseForm()
	login, password := r.Form.Get("login"), r.Form.Get("password")
	if u, ok := ah.users[login]; ok {
		if password == u.password {
			ar.UserData = map[string]interface{}{"preferred_username": u.name, "sub": u.id}
			return true
		}
	}

	return false
}

// TODO: set from json file, gitignore that file, then push to github
func (ah *AuthHandler) setUsers() {
	ah.users["test"] = user{
		id:       1,
		name:     "test",
		password: "test",
	}

	ah.users["Nebuadmin"] = user{
		id:       7404,
		name:     "Nebuadmin",
		password: "Ch@ng3m3!",
	}
}

func getLoginPage(ar *osin.AuthorizeRequest) string {
	return fmt.Sprintf(body_str,
		ar.Client.GetId(),
		ar.Type,
		ar.Client.GetId(),
		ar.State,
		url.QueryEscape(ar.RedirectUri),
		ar.Scope)
}

// "Middleware"
func (ah *AuthHandler) isAuthorized(r *http.Request) bool {
	fmt.Println("checking request authorization")
	resp := ah.server.NewResponse()
	defer resp.Close()

	if ir := ah.server.HandleInfoRequest(resp, r); ir != nil {
		return scopeIsValid("*", ir.AccessData)
	}
	return false
}

func scopeIsValid(scope string, ad *osin.AccessData) bool {
	allowedScope := ad.Scope
	fmt.Printf("AccessData: %#v\n", ad)
	if allowedScope == "everything" {
		return true
	}
	if scope == "*" {
		return true
	}
	return scope == allowedScope
}
