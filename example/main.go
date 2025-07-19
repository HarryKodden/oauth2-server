package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Example demonstrates how to use the OAuth2 flows programmatically
func main() {
	baseURL := "http://localhost:8080"
	
	fmt.Println("🚀 Fosite OAuth2 Flow Example")
	fmt.Println("=============================")
	
	// Example 1: Device Code Flow
	fmt.Println("\n📱 Device Code Flow Example")
	deviceFlowExample(baseURL)
	
	// Example 2: Token Exchange (requires manual authorization)
	fmt.Println("\n🔄 Token Exchange Example")
	fmt.Println("Note: This requires manual user authorization in the device flow")
}

func deviceFlowExample(baseURL string) {
	// Step 1: Initiate device authorization
	fmt.Println("Step 1: Initiating device authorization...")
	
	data := url.Values{}
	data.Set("client_id", "frontend-client")
	data.Set("scope", "openid profile email offline_access")
	
	resp, err := http.PostForm(baseURL+"/device", data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	var deviceAuth map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&deviceAuth)
	
	fmt.Printf("Device Code: %s\n", deviceAuth["device_code"])
	fmt.Printf("User Code: %s\n", deviceAuth["user_code"])
	fmt.Printf("Verification URL: %s\n", deviceAuth["verification_uri_complete"])
	fmt.Printf("Expires in: %.0f seconds\n", deviceAuth["expires_in"])
	
	deviceCode := deviceAuth["device_code"].(string)
	
	// Step 2: Simulate polling (in real scenario, user authorizes in browser)
	fmt.Println("\nStep 2: Polling for token (user must authorize in browser)...")
	fmt.Printf("Please visit: %s\n", deviceAuth["verification_uri_complete"])
	fmt.Println("Username: john.doe, Password: password123")
	fmt.Println("Polling will start in 10 seconds to give you time to authorize...")
	
	time.Sleep(10 * time.Second)
	
	// Poll for token
	for i := 0; i < 12; i++ { // Poll for up to 1 minute
		tokenResp := pollForToken(baseURL, deviceCode)
		if tokenResp != nil {
			if accessToken, ok := tokenResp["access_token"].(string); ok {
				fmt.Printf("✅ Access Token received: %s...\n", accessToken[:20])
				
				if refreshToken, ok := tokenResp["refresh_token"].(string); ok {
					fmt.Printf("✅ Refresh Token received: %s...\n", refreshToken[:20])
				}
				
				// Demonstrate token exchange
				tokenExchangeExample(baseURL, accessToken)
				return
			}
		}
		
		fmt.Printf("⏳ Waiting for authorization... (attempt %d/12)\n", i+1)
		time.Sleep(5 * time.Second)
	}
	
	fmt.Println("❌ Timeout waiting for authorization")
}

func pollForToken(baseURL, deviceCode string) map[string]interface{} {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", deviceCode)
	data.Set("client_id", "frontend-client")
	
	resp, err := http.PostForm(baseURL+"/token", data)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	
	if errorCode, ok := tokenResp["error"].(string); ok {
		if errorCode == "authorization_pending" {
			return nil // Continue polling
		}
		fmt.Printf("Token error: %s\n", errorCode)
		return nil
	}
	
	return tokenResp
}

func tokenExchangeExample(baseURL, subjectToken string) {
	fmt.Println("\n🔄 Token Exchange Example")
	
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	data.Set("client_id", "backend-client")
	data.Set("client_secret", "backend-client-secret")
	data.Set("subject_token", subjectToken)
	data.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	data.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	data.Set("audience", "api-service")
	
	resp, err := http.PostForm(baseURL+"/token", data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	var exchangeResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exchangeResp)
	
	if accessToken, ok := exchangeResp["access_token"].(string); ok {
		fmt.Printf("✅ Exchanged Token: %s...\n", accessToken[:20])
		
		if refreshToken, ok := exchangeResp["refresh_token"].(string); ok {
			fmt.Printf("✅ Refresh Token: %s...\n", refreshToken[:20])
			
			// Demonstrate refresh
			refreshTokenExample(baseURL, refreshToken)
		}
	} else {
		fmt.Printf("❌ Token exchange failed: %v\n", exchangeResp)
	}
}

func refreshTokenExample(baseURL, refreshToken string) {
	fmt.Println("\n🔄 Refresh Token Example")
	
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", "backend-client")
	data.Set("client_secret", "backend-client-secret")
	data.Set("refresh_token", refreshToken)
	
	resp, err := http.PostForm(baseURL+"/token", data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	var refreshResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&refreshResp)
	
	if accessToken, ok := refreshResp["access_token"].(string); ok {
		fmt.Printf("✅ Refreshed Token: %s...\n", accessToken[:20])
	} else {
		fmt.Printf("❌ Token refresh failed: %v\n", refreshResp)
	}
}

// HTTPClient demonstrates how a client application would integrate these flows
type HTTPClient struct {
	baseURL      string
	clientID     string
	clientSecret string
	accessToken  string
	refreshToken string
}

func NewHTTPClient(baseURL, clientID, clientSecret string) *HTTPClient {
	return &HTTPClient{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (c *HTTPClient) DeviceCodeFlow(scopes string) error {
	// Initiate device authorization
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("scope", scopes)
	
	resp, err := http.PostForm(c.baseURL+"/device", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var deviceAuth map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&deviceAuth); err != nil {
		return err
	}
	
	deviceCode := deviceAuth["device_code"].(string)
	
	// In a real application, you would display the user code and verification URL
	// to the user and then poll for the token
	fmt.Printf("Visit: %s\n", deviceAuth["verification_uri_complete"])
	
	// Poll for token (simplified)
	for i := 0; i < 12; i++ {
		if c.pollForDeviceToken(deviceCode) {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for authorization")
}

func (c *HTTPClient) pollForDeviceToken(deviceCode string) bool {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", deviceCode)
	data.Set("client_id", c.clientID)
	
	resp, err := http.PostForm(c.baseURL+"/token", data)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	
	if accessToken, ok := tokenResp["access_token"].(string); ok {
		c.accessToken = accessToken
		if refreshToken, ok := tokenResp["refresh_token"].(string); ok {
			c.refreshToken = refreshToken
		}
		return true
	}
	
	return false
}

func (c *HTTPClient) TokenExchange(subjectToken, audience string) error {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("subject_token", subjectToken)
	data.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	data.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	data.Set("audience", audience)
	
	resp, err := http.PostForm(c.baseURL+"/token", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var exchangeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&exchangeResp); err != nil {
		return err
	}
	
	if accessToken, ok := exchangeResp["access_token"].(string); ok {
		c.accessToken = accessToken
		if refreshToken, ok := exchangeResp["refresh_token"].(string); ok {
			c.refreshToken = refreshToken
		}
		return nil
	}
	
	return fmt.Errorf("token exchange failed")
}

func (c *HTTPClient) RefreshAccessToken() error {
	if c.refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}
	
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("refresh_token", c.refreshToken)
	
	resp, err := http.PostForm(c.baseURL+"/token", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var refreshResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return err
	}
	
	if accessToken, ok := refreshResp["access_token"].(string); ok {
		c.accessToken = accessToken
		if refreshToken, ok := refreshResp["refresh_token"].(string); ok {
			c.refreshToken = refreshToken
		}
		return nil
	}
	
	return fmt.Errorf("token refresh failed")
}

func (c *HTTPClient) MakeAuthenticatedRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	
	// If we get 401, try to refresh the token
	if err == nil && resp.StatusCode == 401 && c.refreshToken != "" {
		resp.Body.Close()
		
		if refreshErr := c.RefreshAccessToken(); refreshErr == nil {
			// Retry with new token
			req.Header.Set("Authorization", "Bearer "+c.accessToken)
			return client.Do(req)
		}
	}
	
	return resp, err
}
