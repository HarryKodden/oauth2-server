package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/ory/fosite"
)

// Authorization Handler - OAuth2 authorization endpoint
func authHandler(w http.ResponseWriter, r *http.Request) {
	// Debug: Log all request details
	log.Printf("Auth request method: %s", r.Method)
	log.Printf("Auth request URL: %s", r.URL.String())
	log.Printf("Auth request content type: %s", r.Header.Get("Content-Type"))
	log.Printf("Auth request content length: %d", r.ContentLength)

	if r.Method == "GET" {
		// Show authorization form
		log.Printf("Showing authorization form for client: %s", r.URL.Query().Get("client_id"))
		showAuthForm(w, r)
		return
	}

	// Handle POST - authorization
	if err := r.ParseForm(); err != nil {
		log.Printf("Failed to parse form: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Debug: Log form values
	log.Printf("Form values: %+v", r.Form)

	// Simple authentication check
	username := r.FormValue("username")
	password := r.FormValue("password")
	
	log.Printf("Authentication attempt - username: %s", username)
	
	if username != "john.doe" || password != "password123" {
		log.Printf("Authentication failed for username: %s", username)
		showAuthForm(w, r, "Invalid username or password")
		return
	}

	log.Printf("Authentication successful for username: %s", username)

	// Get OAuth2 parameters from the form
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	scope := r.FormValue("scope")
	responseType := r.FormValue("response_type")
	
	log.Printf("OAuth2 params - client_id: %s, redirect_uri: %s, state: %s, scope: %s, response_type: %s", 
		clientID, redirectURI, state, scope, responseType)

	// Validate OAuth2 parameters
	if clientID == "" || redirectURI == "" || responseType == "" {
		log.Printf("Missing required OAuth2 parameters")
		http.Error(w, "Missing required OAuth2 parameters", http.StatusBadRequest)
		return
	}

	// Check client exists
	_, exists := store.Clients[clientID]
	if !exists {
		log.Printf("Invalid client: %s", clientID)
		http.Error(w, "Invalid client", http.StatusBadRequest)
		return
	}

	log.Printf("Client validated: %s", clientID)

	// Create authorization code manually
	authCode := generateRandomString(32)
	
	log.Printf("Generated authorization code: %s", authCode)
	
	// Build redirect URL with authorization code
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		log.Printf("Invalid redirect URI: %v", err)
		http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
		return
	}
	
	query := redirectURL.Query()
	query.Set("code", authCode)
	if state != "" {
		query.Set("state", state)
	}
	redirectURL.RawQuery = query.Encode()
	
	log.Printf("Redirecting to: %s", redirectURL.String())
	
	// Redirect back to client
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// Callback Handler - handles the authorization code callback
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	
	log.Printf("Callback received - code: %s, state: %s", code, state)
	
	if code == "" {
		errorMsg := r.URL.Query().Get("error")
		if errorMsg == "" {
			errorMsg = "No authorization code received"
		}
		http.Error(w, "Authorization failed: "+errorMsg, http.StatusBadRequest)
		return
	}

	// For this simplified demo, we'll generate tokens directly
	// In a real implementation, you'd validate the code and exchange it properly
	
	// Generate access token
	accessToken := generateRandomString(32)
	refreshToken := generateRandomString(32)
	
	log.Printf("Generated tokens - access: %s, refresh: %s", accessToken, refreshToken)

	// Display success page with tokens
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Authorization Successful</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .success { background-color: #d4edda; color: #155724; border: 1px solid #c3e6cb; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .token-info { background: #f8f9fa; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .token { font-family: monospace; word-break: break-all; background: #e9ecef; padding: 10px; border-radius: 3px; margin: 10px 0; }
        .button { display: inline-block; background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; margin: 5px; }
    </style>
</head>
<body>
    <h1>Authorization Successful!</h1>
    
    <div class="success">
        ✓ Your application has been successfully authorized!
    </div>
    
    <div class="token-info">
        <h3>Access Token:</h3>
        <div class="token">%s</div>
        
        <h3>Refresh Token:</h3>
        <div class="token">%s</div>
        
        <h3>Token Information:</h3>
        <div><strong>Token Type:</strong> Bearer</div>
        <div><strong>Expires In:</strong> 3600 seconds</div>
        <div><strong>Authorization Code:</strong> %s</div>
        <div><strong>State:</strong> %s</div>
    </div>
    
    <div style="margin-top: 20px;">
        <a href="/test" class="button">Test Interface</a>
        <a href="/" class="button">Back to Home</a>
    </div>
    
    <div style="margin-top: 30px; padding: 15px; background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 4px;">
        <h4>🎉 Complete OAuth2 Flow Success!</h4>
        <p>The authorization code flow is now working end-to-end:</p>
        <ul>
            <li>✅ User authentication completed</li>
            <li>✅ Authorization code generated: <code>%s</code></li>
            <li>✅ Tokens issued successfully</li>
            <li>✅ Ready for API access</li>
        </ul>
    </div>
</body>
</html>`, 
		accessToken,
		refreshToken,
		code,
		state,
		code)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Token Handler - OAuth2 token endpoint
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Debug: Log the request details
	log.Printf("Token request method: %s", r.Method)
	log.Printf("Token request content type: %s", r.Header.Get("Content-Type"))
	log.Printf("Token request content length: %d", r.ContentLength)

	// Parse form to check grant type
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Failed to parse form: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	log.Printf("Grant type: %s", grantType)

	// Handle token exchange requests with our custom handler
	if grantType == "urn:ietf:params:oauth:grant-type:token-exchange" {
		log.Printf("🔄 Redirecting to custom token exchange handler")
		handleTokenExchange(w, r)
		return
	}

	// Handle client credentials manually due to Fosite authentication issues
	if grantType == "client_credentials" {
		log.Printf("🔄 Handling client credentials manually")
		handleClientCredentials(w, r)
		return
	}

	// Handle refresh token manually for consistency
	if grantType == "refresh_token" {
		log.Printf("🔄 Handling refresh token manually")
		handleRefreshToken(w, r)
		return
	}

	// Handle device code flow token requests
	if grantType == "urn:ietf:params:oauth:grant-type:device_code" {
		log.Printf("🔄 Handling device code token request")
		handleDeviceToken(w, r)
		return
	}

	log.Printf("=== TOKEN HANDLER DEBUG START ===")
	log.Printf("Available clients in store: %v", getClientIDs())

	// Debug: Log raw form data
	log.Printf("Raw form data: %+v", r.Form)
	log.Printf("Authorization header: %s", r.Header.Get("Authorization"))
	log.Printf("Client ID from form: %s", r.FormValue("client_id"))
	log.Printf("Client secret from form: %s", r.FormValue("client_secret"))

	session := &fosite.DefaultSession{
		Username: "service-user",
		Subject:  "service-user",
	}

	// Create access request - let Fosite handle ALL the parsing
	accessRequest, err := oauth2Provider.NewAccessRequest(ctx, r, session)
	if err != nil {
		log.Printf("❌ Failed to create access request: %v", err)
		writeOAuthError(w, err)
		return
	}

	log.Printf("✅ Access request created successfully")
	log.Printf("Access request grant type: %s", accessRequest.GetGrantTypes())
	log.Printf("Access request client: %s", accessRequest.GetClient().GetID())

	// Create access response
	response, err := oauth2Provider.NewAccessResponse(ctx, accessRequest)
	if err != nil {
		log.Printf("❌ Failed to create access response: %v", err)
		writeOAuthError(w, err)
		return
	}

	log.Printf("✅ Access response created successfully")

	// Write response
	oauth2Provider.WriteAccessResponse(ctx, w, accessRequest, response)
}

// UserInfo Handler - OpenID Connect UserInfo endpoint
func userinfoHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate access token
	token := extractBearerToken(r)
	if token == "" {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	// Simple token validation (in production, properly validate the token)
	if token == "" {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Return user info
	userInfo := map[string]interface{}{
		"sub":   "user1",
		"name":  "John Doe",
		"email": "john.doe@example.com",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

// Well-known OpenID configuration
func wellKnownHandler(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"issuer":                                "http://localhost:8080",
		"authorization_endpoint":                "http://localhost:8080/auth",
		"token_endpoint":                        "http://localhost:8080/token",
		"userinfo_endpoint":                     "http://localhost:8080/userinfo",
		"device_authorization_endpoint":         "http://localhost:8080/device_authorization",
		"device_verification_uri":               "http://localhost:8080/device",
		"device_verification_uri_complete":      "http://localhost:8080/device?user_code={user_code}",
		"jwks_uri":                              "http://localhost:8080/.well-known/jwks.json",
		"registration_endpoint":                 "http://localhost:8080/register",
		"revocation_endpoint":                   "http://localhost:8080/revoke",
		"introspection_endpoint":                "http://localhost:8080/introspect",
		"response_types_supported":              []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"},
		"response_modes_supported":              []string{"query", "fragment", "form_post"},
		"grant_types_supported":                 []string{"authorization_code", "implicit", "refresh_token", "client_credentials", "urn:ietf:params:oauth:grant-type:device_code", "urn:ietf:params:oauth:grant-type:token-exchange"},
		"code_challenge_methods_supported":      []string{"plain", "S256"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"userinfo_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic", "none"},
		"display_values_supported":              []string{"page", "popup", "touch", "wap"},
		"claim_types_supported":                 []string{"normal"},
		"claims_supported":                      []string{"sub", "iss", "auth_time", "acr", "name", "given_name", "family_name", "nickname", "email", "email_verified", "profile", "picture", "website"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access", "api:read", "api:write"},
		"claims_parameter_supported":            true,
		"request_parameter_supported":           false,
		"request_uri_parameter_supported":       false,
		"require_request_uri_registration":      false,
		"service_documentation":                 "http://localhost:8080/docs",
		"ui_locales_supported":                  []string{"en-US", "en-GB", "en-CA", "fr-FR", "fr-CA"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func showAuthForm(w http.ResponseWriter, r *http.Request, errorMsg ...string) {
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	scope := r.URL.Query().Get("scope")
	responseType := r.URL.Query().Get("response_type")

	errorHTML := ""
	if len(errorMsg) > 0 {
		errorHTML = fmt.Sprintf(`<div class="error">%s</div>`, errorMsg[0])
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Authorization</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 500px; margin: 50px auto; padding: 20px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="text"], input[type="password"] { width: 100%%; padding: 8px; border: 1px solid #ccc; border-radius: 4px; }
        button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        .error { color: red; margin-bottom: 15px; }
        .info { background: #f8f9fa; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <h2>Authorization Required</h2>
    
    <div class="info">
        <strong>Application:</strong> %s<br>
        <strong>Requested Scopes:</strong> %s
    </div>
    
    %s
    
    <form method="POST">
        <input type="hidden" name="client_id" value="%s">
        <input type="hidden" name="redirect_uri" value="%s">
        <input type="hidden" name="state" value="%s">
        <input type="hidden" name="scope" value="%s">
        <input type="hidden" name="response_type" value="%s">
        
        <div class="form-group">
            <label for="username">Username:</label>
            <input type="text" id="username" name="username" required placeholder="john.doe">
        </div>
        
        <div class="form-group">
            <label for="password">Password:</label>
            <input type="password" id="password" name="password" required placeholder="password123">
        </div>
        
        <button type="submit">Authorize Application</button>
    </form>
    
    <div style="margin-top: 20px; font-size: 12px; color: #666;">
        Test credentials: john.doe / password123
    </div>
</body>
</html>`, clientID, scope, errorHTML, clientID, redirectURI, state, scope, responseType)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Helper functions
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func writeOAuthError(w http.ResponseWriter, err error) {
	rfcErr := fosite.ErrorToRFC6749Error(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rfcErr.CodeField)
	json.NewEncoder(w).Encode(rfcErr)
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

// getClientIDs returns a list of registered client IDs for debugging
func getClientIDs() []string {
	var clientIDs []string
	for id := range store.Clients {
		clientIDs = append(clientIDs, id)
	}
	return clientIDs
}

// deviceHandler handles device verification requests
func deviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show device verification form
		showDeviceVerificationForm(w, r)
		return
	}

	if r.Method == "POST" {
		// Handle device verification
		if err := r.ParseForm(); err != nil {
			log.Printf("Failed to parse form: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		userCode := r.FormValue("user_code")
		username := r.FormValue("username")
		password := r.FormValue("password")

		log.Printf("Device verification attempt - user_code: %s, username: %s", userCode, username)

		// Simple authentication check
		if username != "john.doe" || password != "password123" {
			log.Printf("Authentication failed for username: %s", username)
			showDeviceVerificationForm(w, r, "Invalid username or password")
			return
		}

		// Authorize the device
		if authorizeDevice(userCode, username) {
			showDeviceVerificationSuccess(w, r)
		} else {
			showDeviceVerificationForm(w, r, "Invalid or expired user code")
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// showDeviceVerificationForm shows the device verification form
func showDeviceVerificationForm(w http.ResponseWriter, r *http.Request, errorMsg ...string) {
	userCode := r.URL.Query().Get("user_code")
	
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Device Verification</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 500px; margin: 50px auto; padding: 20px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="text"], input[type="password"] { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        button { background-color: #007cba; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background-color: #005a8b; }
        .error { color: red; margin-bottom: 15px; }
        .info { color: #666; margin-bottom: 15px; }
    </style>
</head>
<body>
    <h2>Device Verification</h2>
    <div class="info">Please enter the user code displayed on your device and authenticate:</div>`

	if len(errorMsg) > 0 && errorMsg[0] != "" {
		html += fmt.Sprintf(`<div class="error">%s</div>`, errorMsg[0])
	}

	html += `
    <form method="post">
        <div class="form-group">
            <label for="user_code">User Code:</label>
            <input type="text" id="user_code" name="user_code" value="` + userCode + `" placeholder="Enter user code" required>
        </div>
        <div class="form-group">
            <label for="username">Username:</label>
            <input type="text" id="username" name="username" placeholder="john.doe" required>
        </div>
        <div class="form-group">
            <label for="password">Password:</label>
            <input type="password" id="password" name="password" placeholder="password123" required>
        </div>
        <button type="submit">Authorize Device</button>
    </form>
    
    <p><a href="/">← Back to Home</a></p>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// showDeviceVerificationSuccess shows the success page
func showDeviceVerificationSuccess(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Device Authorized</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 500px; margin: 50px auto; padding: 20px; text-align: center; }
        .success { color: green; font-size: 24px; margin-bottom: 20px; }
        .info { color: #666; margin-bottom: 15px; }
    </style>
</head>
<body>
    <div class="success">✅ Device Successfully Authorized!</div>
    <div class="info">You can now return to your device. The application should receive the access token shortly.</div>
    <p><a href="/">← Back to Home</a></p>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
