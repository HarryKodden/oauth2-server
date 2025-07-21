package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"
)

type DeviceAuthResponse struct {
    DeviceCode              string `json:"device_code"`
    UserCode                string `json:"user_code"`
    VerificationURI         string `json:"verification_uri"`
    VerificationURIComplete string `json:"verification_uri_complete"`
    ExpiresIn               int    `json:"expires_in"`
    Interval                int    `json:"interval"`
}

type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token"`
    Scope        string `json:"scope"`
}

type ErrorResponse struct {
    Error            string `json:"error"`
    ErrorDescription string `json:"error_description"`
}

func main() {
    baseURL := "http://localhost:8080"
    clientID := "smart-tv-app"

    fmt.Println("🚀 Starting OAuth2 Device Flow Demo")
    fmt.Println("===================================")

    // Step 1: Request device authorization
    fmt.Println("\n📱 Step 1: Requesting device authorization...")
    
    data := url.Values{}
    data.Set("client_id", clientID)
    data.Set("scope", "openid profile api:read")

    resp, err := http.PostForm(baseURL+"/device_authorization", data)
    if err != nil {
        fmt.Printf("❌ Error requesting device authorization: %v\n", err)
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    
    if resp.StatusCode != 200 {
        fmt.Printf("❌ Device authorization failed: %s\n", string(body))
        return
    }

    var deviceAuth DeviceAuthResponse
    if err := json.Unmarshal(body, &deviceAuth); err != nil {
        fmt.Printf("❌ Error parsing device auth response: %v\n", err)
        return
    }

    fmt.Printf("✅ Device authorization received!\n")
    fmt.Printf("   📋 User Code: %s\n", deviceAuth.UserCode)
    fmt.Printf("   🌐 Verification URI: %s\n", deviceAuth.VerificationURI)
    fmt.Printf("   🔗 Complete URI: %s\n", deviceAuth.VerificationURIComplete)
    fmt.Printf("   ⏰ Expires in: %d seconds\n", deviceAuth.ExpiresIn)
    fmt.Printf("   🔄 Polling interval: %d seconds\n", deviceAuth.Interval)

    fmt.Println("\n🔐 Step 2: User Authorization Required")
    fmt.Println("=====================================")
    fmt.Printf("Please visit: %s\n", deviceAuth.VerificationURIComplete)
    fmt.Printf("Or go to: %s and enter code: %s\n", deviceAuth.VerificationURI, deviceAuth.UserCode)
    fmt.Println("\nTest credentials:")
    fmt.Println("  Username: john.doe, Password: password123")
    fmt.Println("  Username: admin, Password: admin123")

    // Step 3: Poll for token
    fmt.Println("\n⏳ Step 3: Polling for access token...")
    fmt.Println("Waiting for user authorization...")

    ticker := time.NewTicker(time.Duration(deviceAuth.Interval) * time.Second)
    defer ticker.Stop()

    timeout := time.NewTimer(time.Duration(deviceAuth.ExpiresIn) * time.Second)
    defer timeout.Stop()

    for {
        select {
        case <-timeout.C:
            fmt.Println("❌ Device authorization expired")
            return

        case <-ticker.C:
            fmt.Print(".")
            
            tokenData := url.Values{}
            tokenData.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
            tokenData.Set("device_code", deviceAuth.DeviceCode)
            tokenData.Set("client_id", clientID)

            tokenResp, err := http.PostForm(baseURL+"/token", tokenData)
            if err != nil {
                fmt.Printf("\n❌ Error polling for token: %v\n", err)
                continue
            }

            tokenBody, _ := io.ReadAll(tokenResp.Body)
            tokenResp.Body.Close()

            if tokenResp.StatusCode == 200 {
                var token TokenResponse
                if err := json.Unmarshal(tokenBody, &token); err != nil {
                    fmt.Printf("\n❌ Error parsing token response: %v\n", err)
                    continue
                }

                fmt.Printf("\n🎉 SUCCESS! Access token received!\n")
                fmt.Printf("   🔑 Access Token: %s...\n", token.AccessToken[:20])
                fmt.Printf("   🔄 Refresh Token: %s...\n", token.RefreshToken[:20])
                fmt.Printf("   📝 Scope: %s\n", token.Scope)
                fmt.Printf("   ⏰ Expires in: %d seconds\n", token.ExpiresIn)
                return

            } else {
                var errorResp ErrorResponse
                json.Unmarshal(tokenBody, &errorResp)

                if errorResp.Error == "authorization_pending" {
                    // Continue polling
                    continue
                } else {
                    fmt.Printf("\n❌ Token error: %s - %s\n", errorResp.Error, errorResp.ErrorDescription)
                    return
                }
            }
        }
    }
}