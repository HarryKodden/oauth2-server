package main

import (
	"net/http"
)

// Sample application handlers

// homeHandler shows the main page with links to sample clients
func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Fosite OAuth2 Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .client-section { background: #f5f5f5; padding: 20px; margin: 20px 0; border-radius: 5px; }
        .button { display: inline-block; background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; margin: 5px; }
        .button:hover { background-color: #0056b3; }
        .endpoint { background: #e9ecef; padding: 10px; margin: 10px 0; border-radius: 3px; font-family: monospace; }
    </style>
</head>
<body>
    <h1>Fosite OAuth2 Example Server</h1>
    
    <div class="client-section">
        <h2>Client 1: Authorization Code Flow</h2>
        <p>This client demonstrates the OAuth2 Authorization Code Flow. 
           It simulates a device-like flow but uses standard OAuth2.</p>
        <a href="/client1/auth" class="button">Start Authorization Flow</a>
        
        <h3>Supported Features:</h3>
        <ul>
            <li>Authorization Code Flow</li>
            <li>Refresh Tokens</li>
            <li>OpenID Connect scopes</li>
            <li>Audience-based tokens</li>
        </ul>
    </div>
    
    <div class="client-section">
        <h2>📱 Device Code Flow (RFC 8628)</h2>
        <p>Perfect for CLI applications and IoT devices! This flow allows users to authorize 
           devices that can't easily display a web browser.</p>
        <a href="/device-flow-demo" class="button">Start Device Flow Demo</a>
        
        <h3>How it works:</h3>
        <ol>
            <li>Device requests a device code and user code</li>
            <li>User visits verification URL and enters user code</li>
            <li>Device polls for authorization</li>
            <li>Tokens are issued once user authorizes</li>
        </ol>
        
        <h3>Supported Features:</h3>
        <ul>
            <li>Device Authorization (RFC 8628)</li>
            <li>User Code Verification</li>
            <li>Device Token Polling</li>
            <li>Refresh Tokens</li>
        </ul>
    </div>
    
    <div class="client-section">
        <h2>Client 2: Client Credentials & Token Exchange</h2>
        <p>This client demonstrates Client Credentials flow and token exchange concepts. 
           It can use tokens for service-specific operations.</p>
        <a href="/client2/exchange" class="button">Client Credentials Demo</a>
        
        <h3>Supported Features:</h3>
        <ul>
            <li>Client Credentials Flow</li>
            <li>Refresh Tokens</li>
            <li>Service-to-service authentication</li>
            <li>Token validation</li>
        </ul>
    </div>
    
    <div class="client-section">
        <h2>🧪 Complete Flow Tester</h2>
        <p>Interactive testing interface for the complete OAuth2 flow including both clients.</p>
        <a href="/test" class="button">Open Flow Tester</a>
    </div>
    
    <div class="client-section">
        <h2>OAuth2 Endpoints</h2>
        <div class="endpoint">Authorization: GET http://localhost:8080/auth</div>
        <div class="endpoint">Token Endpoint: POST http://localhost:8080/token</div>
        <div class="endpoint">Device Authorization: POST http://localhost:8080/device_authorization</div>
        <div class="endpoint">Device Verification: GET http://localhost:8080/device</div>
        <div class="endpoint">UserInfo: GET http://localhost:8080/userinfo</div>
        <div class="endpoint">OpenID Configuration: GET http://localhost:8080/.well-known/openid_configuration</div>
    </div>
    
    <div class="client-section">
        <h2>Test Credentials</h2>
        <p><strong>Username:</strong> john.doe</p>
        <p><strong>Password:</strong> password123</p>
    </div>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// client1AuthFlowHandler demonstrates the authorization code flow
func client1AuthFlowHandler(w http.ResponseWriter, r *http.Request) {
	// Redirect to authorization endpoint
	authURL := "http://localhost:8080/auth?" +
		"client_id=frontend-client&" +
		"redirect_uri=http://localhost:8080/callback&" +
		"response_type=code&" +
		"scope=openid+profile+email+offline_access&" +
		"state=random-state-123"

	http.Redirect(w, r, authURL, http.StatusFound)
}

// client2TokenExchangeHandler demonstrates client credentials flow
func client2TokenExchangeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Client Credentials Flow - Client 2</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="text"], textarea { width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px; }
        textarea { height: 100px; font-family: monospace; font-size: 12px; }
        button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        .result { margin: 20px 0; padding: 15px; border-radius: 4px; }
        .success { background-color: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .error { background-color: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .code { font-family: monospace; background: #f8f9fa; padding: 10px; border-radius: 3px; white-space: pre-wrap; word-break: break-all; }
    </style>
    <script>
        function getClientToken() {
            const data = new FormData();
            data.append('grant_type', 'client_credentials');
            data.append('client_id', 'backend-client');
            data.append('client_secret', 'backend-client-secret');
            data.append('scope', 'api:read api:write');
            
            fetch('/token', {
                method: 'POST',
                body: data
            })
            .then(response => response.json())
            .then(data => {
                if (data.access_token) {
                    showResult('Client credentials flow successful!\\n\\n' + JSON.stringify(data, null, 2), 'success');
                    
                    // Store the token for potential refresh demonstration
                    window.serviceToken = data;
                } else {
                    showResult('Error: ' + (data.error_description || data.error), 'error');
                }
            })
            .catch(error => {
                showResult('Network error: ' + error.message, 'error');
            });
        }
        
        function testTokenValidation() {
            if (!window.serviceToken || !window.serviceToken.access_token) {
                showResult('No token available. Get a token first.', 'error');
                return;
            }
            
            fetch('/userinfo', {
                method: 'GET',
                headers: {
                    'Authorization': 'Bearer ' + window.serviceToken.access_token
                }
            })
            .then(response => response.json())
            .then(data => {
                if (data.sub) {
                    showResult('Token validation successful!\\n\\n' + JSON.stringify(data, null, 2), 'success');
                } else {
                    showResult('Token validation failed: ' + (data.error_description || data.error || 'Unknown error'), 'error');
                }
            })
            .catch(error => {
                showResult('Network error: ' + error.message, 'error');
            });
        }
        
        function showResult(message, type) {
            const resultDiv = document.getElementById('result');
            resultDiv.className = 'result ' + type;
            resultDiv.innerHTML = '<div class="code">' + message + '</div>';
        }
    </script>
</head>
<body>
    <h1>Client Credentials Flow - Client 2</h1>
    
    <p>This client demonstrates the OAuth2 Client Credentials flow. This is perfect for 
       service-to-service authentication where no user interaction is required.</p>
    
    <div style="margin: 20px 0;">
        <button onclick="getClientToken()">Get Service Token</button>
        <button onclick="testTokenValidation()" style="margin-left: 10px;">Test Token</button>
    </div>
    
    <div id="result"></div>
    
    <div style="margin-top: 30px; padding: 20px; background: #f8f9fa; border-radius: 5px;">
        <h3>Client 2 Configuration</h3>
        <div><strong>Client ID:</strong> backend-client</div>
        <div><strong>Grant Types:</strong> client_credentials, refresh_token</div>
        <div><strong>Scopes:</strong> api:read, api:write</div>
        <div><strong>Use Case:</strong> Service-to-service authentication</div>
        
        <h4 style="margin-top: 20px;">How this simulates your requirements:</h4>
        <ul>
            <li><strong>Long-running process:</strong> Client credentials are perfect for background services</li>
            <li><strong>Token refresh:</strong> Service can automatically refresh tokens without user interaction</li>
            <li><strong>Audience scoping:</strong> Tokens are scoped to specific API services</li>
            <li><strong>No user context:</strong> Operates independently of user sessions</li>
        </ul>
    </div>
    
    <div style="margin-top: 20px;">
        <a href="/">← Back to Home</a>
    </div>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// serveTestPage serves the interactive test page
func serveTestPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>OAuth2 Flow Tester</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 20px auto; padding: 20px; }
        .test-section { border: 1px solid #ddd; margin: 20px 0; padding: 20px; border-radius: 5px; }
        .success { background-color: #d4edda; color: #155724; border: 1px solid #c3e6cb; padding: 10px; border-radius: 4px; margin: 10px 0; }
        .error { background-color: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; padding: 10px; border-radius: 4px; margin: 10px 0; }
        .info { background-color: #cce5ff; color: #004085; border: 1px solid #b8daff; padding: 10px; border-radius: 4px; margin: 10px 0; }
        button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        button:hover { background-color: #0056b3; }
        textarea { width: 100%; height: 150px; padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-family: monospace; }
        .token-display { background: #f8f9fa; padding: 15px; border-radius: 4px; margin: 10px 0; word-break: break-all; font-family: monospace; }
    </style>
</head>
<body>
    <h1>🧪 OAuth2 Flow Complete Tester</h1>
    <div class="info">
        <strong>Test Scenario:</strong> This tests the complete flow where Client1 (frontend-client) authenticates a user, 
        and Client2 (backend-client) can use tokens for API access with refresh capability.
    </div>

    <!-- Step 1: Client2 Client Credentials -->
    <div class="test-section">
        <h3>Step 1: Test Client2 (Service Client) - Client Credentials Flow</h3>
        <p>This tests that the service client can get its own access token using client credentials.</p>
        <button onclick="testClient2Credentials()">Test Client2 Client Credentials</button>
        <div id="client2-result"></div>
    </div>

    <!-- Step 2: Client1 Authorization -->
    <div class="test-section">
        <h3>Step 2: Test Client1 (Device Client) - Authorization Code Flow</h3>
        <p><strong>Manual Steps:</strong></p>
        <ol>
            <li>Click the button below to start the authorization flow</li>
            <li>Login with: <code>john.doe</code> / <code>password123</code></li>
            <li>Copy the callback URL and paste it below</li>
        </ol>
        <button onclick="startClient1Auth()">Start Client1 Authorization</button>
        <div style="margin: 10px 0;">
            <label for="callback-url"><strong>Callback URL:</strong></label>
            <input type="text" id="callback-url" style="width: 100%; padding: 8px; margin: 5px 0;" 
                   placeholder="http://localhost:8080/callback?code=...&state=...">
            <button onclick="exchangeCodeForToken()">Exchange Code for Token</button>
        </div>
        <div id="client1-result"></div>
    </div>

    <!-- Results -->
    <div class="test-section">
        <h3>Test Results</h3>
        <textarea id="test-log" readonly placeholder="Test results will appear here..."></textarea>
        <button onclick="clearLog()">Clear Log</button>
    </div>

    <script>
        function log(message) {
            const logArea = document.getElementById('test-log');
            const timestamp = new Date().toLocaleTimeString();
            logArea.value += '[' + timestamp + '] ' + message + '\n';
            logArea.scrollTop = logArea.scrollHeight;
        }

        function displayResult(elementId, content, isError = false) {
            const element = document.getElementById(elementId);
            element.innerHTML = '<div class="' + (isError ? 'error' : 'success') + '">' + content + '</div>';
        }

        async function testClient2Credentials() {
            log('Starting Client2 client credentials test...');
            try {
                const response = await fetch('/token', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                    body: 'grant_type=client_credentials&client_id=backend-client&client_secret=backend-client-secret&scope=api:read api:write'
                });

                const result = await response.json();
                if (response.ok) {
                    displayResult('client2-result', 
                        '<strong>✓ Client2 Success!</strong><br>' +
                        '<div class="token-display">' +
                        '<strong>Access Token:</strong> ' + result.access_token + '<br>' +
                        '<strong>Token Type:</strong> ' + result.token_type + '<br>' +
                        '<strong>Expires In:</strong> ' + result.expires_in + ' seconds' +
                        '</div>'
                    );
                    log('Client2 success: ' + JSON.stringify(result));
                } else {
                    displayResult('client2-result', '<strong>✗ Failed:</strong> ' + JSON.stringify(result), true);
                    log('Client2 failed: ' + JSON.stringify(result));
                }
            } catch (error) {
                displayResult('client2-result', '<strong>✗ Error:</strong> ' + error.message, true);
                log('Client2 error: ' + error.message);
            }
        }

        function startClient1Auth() {
            const authUrl = '/auth?client_id=frontend-client&redirect_uri=' + 
                          encodeURIComponent('http://localhost:8080/callback') + 
                          '&response_type=code&scope=' + 
                          encodeURIComponent('openid profile email api:read') + 
                          '&state=test123';
            log('Starting Client1 authorization: ' + authUrl);
            window.open(authUrl, '_blank');
            displayResult('client1-result', 
                '<div class="info"><strong>Authorization started!</strong><br>' +
                'Complete the login in the new window, then return here and paste the callback URL.</div>'
            );
        }

        async function exchangeCodeForToken() {
            const callbackUrl = document.getElementById('callback-url').value;
            if (!callbackUrl) {
                displayResult('client1-result', '<strong>Please paste the callback URL first!</strong>', true);
                return;
            }

            try {
                const url = new URL(callbackUrl);
                const code = url.searchParams.get('code');

                if (!code) {
                    displayResult('client1-result', '<strong>No authorization code found in URL!</strong>', true);
                    return;
                }

                log('Exchanging code for token: ' + code);

                const response = await fetch('/token', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                    body: 'grant_type=authorization_code&client_id=frontend-client&client_secret=frontend-client-secret&code=' + 
                          code + '&redirect_uri=' + encodeURIComponent('http://localhost:8080/callback')
                });

                const result = await response.json();
                if (response.ok) {
                    displayResult('client1-result', 
                        '<strong>✓ Client1 Token Exchange Success!</strong><br>' +
                        '<div class="token-display">' +
                        '<strong>Access Token:</strong> ' + result.access_token + '<br>' +
                        '<strong>Token Type:</strong> ' + result.token_type + '<br>' +
                        '<strong>Expires In:</strong> ' + result.expires_in + ' seconds<br>' +
                        (result.refresh_token ? '<strong>Refresh Token:</strong> ' + result.refresh_token : '') +
                        '</div>'
                    );
                    log('Client1 token exchange success: ' + JSON.stringify(result));
                } else {
                    displayResult('client1-result', '<strong>✗ Token Exchange Failed:</strong> ' + JSON.stringify(result), true);
                    log('Client1 token exchange failed: ' + JSON.stringify(result));
                }
            } catch (error) {
                displayResult('client1-result', '<strong>✗ Error:</strong> ' + error.message, true);
                log('Client1 token exchange error: ' + error.message);
            }
        }

        function clearLog() {
            document.getElementById('test-log').value = '';
        }

        // Initialize
        log('OAuth2 Flow Tester loaded. Ready to test!');
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// deviceFlowDemoHandler demonstrates the device code flow
func deviceFlowDemoHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Device Code Flow Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .step { background: #f8f9fa; padding: 20px; margin: 20px 0; border-radius: 5px; border-left: 4px solid #007bff; }
        .button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        .button:hover { background-color: #0056b3; }
        .code { background: #e9ecef; padding: 10px; margin: 10px 0; border-radius: 3px; font-family: monospace; white-space: pre-wrap; }
        .result { background: #d4edda; padding: 15px; margin: 10px 0; border-radius: 3px; border: 1px solid #c3e6cb; }
        .error { background: #f8d7da; padding: 15px; margin: 10px 0; border-radius: 3px; border: 1px solid #f5c6cb; }
        .user-code { font-size: 24px; font-weight: bold; color: #007bff; }
        .polling-status { padding: 10px; margin: 10px 0; border-radius: 3px; }
        .polling { background: #fff3cd; border: 1px solid #ffeaa7; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; }
    </style>
</head>
<body>
    <h1>📱 Device Code Flow Demo (RFC 8628)</h1>
    <p>This demonstrates the OAuth2 Device Authorization Grant, perfect for CLI applications and IoT devices.</p>
    
    <div class="step">
        <h3>Step 1: Start Device Authorization</h3>
        <p>Click the button below to initiate the device code flow. This simulates what a CLI application would do.</p>
        <button onclick="startDeviceFlow()" class="button">Start Device Authorization</button>
        <div id="device-result"></div>
    </div>
    
    <div class="step">
        <h3>Step 2: User Authorization</h3>
        <p>After receiving the device code, the user needs to visit the verification URL and enter the user code.</p>
        <div id="verification-instructions" style="display: none;">
            <p><strong>Instructions for user:</strong></p>
            <ol>
                <li>Open this URL in your browser: <a id="verification-url" href="#" target="_blank"></a></li>
                <li>Enter this user code: <span id="user-code-display" class="user-code"></span></li>
                <li>Login with username: <code>john.doe</code> and password: <code>password123</code></li>
            </ol>
        </div>
    </div>
    
    <div class="step">
        <h3>Step 3: Token Polling</h3>
        <p>The device polls the token endpoint until the user authorizes the device.</p>
        <button onclick="startPolling()" id="poll-button" class="button" disabled>Start Polling for Tokens</button>
        <div id="polling-status"></div>
        <div id="token-result"></div>
    </div>
    
    <div class="step">
        <h3>Step 4: Use Access Token</h3>
        <p>Once authorized, use the access token to make API calls.</p>
        <button onclick="testAccessToken()" id="test-token-button" class="button" disabled>Test Access Token</button>
        <div id="userinfo-result"></div>
    </div>
    
    <p><a href="/">← Back to Home</a></p>

    <script>
        let deviceCode = '';
        let accessToken = '';
        let pollingInterval = null;
        
        async function startDeviceFlow() {
            try {
                const response = await fetch('/device_authorization', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: 'client_id=frontend-client&scope=openid profile email api:read'
                });
                
                const result = await response.json();
                
                if (response.ok) {
                    deviceCode = result.device_code;
                    document.getElementById('device-result').innerHTML = 
                        '<div class="result">' +
                        '<strong>✓ Device Authorization Started!</strong><br>' +
                        '<div class="code">' + JSON.stringify(result, null, 2) + '</div>' +
                        '</div>';
                    
                    // Show verification instructions
                    document.getElementById('verification-url').href = result.verification_uri;
                    document.getElementById('verification-url').textContent = result.verification_uri;
                    document.getElementById('user-code-display').textContent = result.user_code;
                    document.getElementById('verification-instructions').style.display = 'block';
                    document.getElementById('poll-button').disabled = false;
                } else {
                    document.getElementById('device-result').innerHTML = 
                        '<div class="error"><strong>✗ Error:</strong> ' + JSON.stringify(result) + '</div>';
                }
            } catch (error) {
                document.getElementById('device-result').innerHTML = 
                    '<div class="error"><strong>✗ Error:</strong> ' + error.message + '</div>';
            }
        }
        
        async function startPolling() {
            if (!deviceCode) {
                alert('Please start device authorization first');
                return;
            }
            
            document.getElementById('poll-button').disabled = true;
            document.getElementById('polling-status').innerHTML = 
                '<div class="polling-status polling">⏳ Polling for authorization... (waiting for user to authorize)</div>';
            
            pollingInterval = setInterval(async () => {
                try {
                    const response = await fetch('/token', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/x-www-form-urlencoded',
                        },
                        body: 'grant_type=urn:ietf:params:oauth:grant-type:device_code&client_id=frontend-client&device_code=' + deviceCode
                    });
                    
                    const result = await response.json();
                    
                    if (response.ok) {
                        // Success!
                        clearInterval(pollingInterval);
                        accessToken = result.access_token;
                        document.getElementById('polling-status').innerHTML = 
                            '<div class="polling-status success">✅ Authorization successful! Tokens received.</div>';
                        document.getElementById('token-result').innerHTML = 
                            '<div class="result">' +
                            '<strong>✓ Tokens Received!</strong><br>' +
                            '<div class="code">' + JSON.stringify(result, null, 2) + '</div>' +
                            '</div>';
                        document.getElementById('test-token-button').disabled = false;
                    } else if (result.error === 'authorization_pending') {
                        // Still waiting
                        console.log('Still waiting for authorization...');
                    } else {
                        // Error
                        clearInterval(pollingInterval);
                        document.getElementById('polling-status').innerHTML = 
                            '<div class="error"><strong>✗ Error:</strong> ' + result.error_description || result.error + '</div>';
                        document.getElementById('poll-button').disabled = false;
                    }
                } catch (error) {
                    clearInterval(pollingInterval);
                    document.getElementById('polling-status').innerHTML = 
                        '<div class="error"><strong>✗ Error:</strong> ' + error.message + '</div>';
                    document.getElementById('poll-button').disabled = false;
                }
            }, 5000); // Poll every 5 seconds
        }
        
        async function testAccessToken() {
            if (!accessToken) {
                alert('No access token available');
                return;
            }
            
            try {
                const response = await fetch('/userinfo', {
                    headers: {
                        'Authorization': 'Bearer ' + accessToken
                    }
                });
                
                const result = await response.json();
                
                if (response.ok) {
                    document.getElementById('userinfo-result').innerHTML = 
                        '<div class="result">' +
                        '<strong>✓ UserInfo API Success!</strong><br>' +
                        '<div class="code">' + JSON.stringify(result, null, 2) + '</div>' +
                        '</div>';
                } else {
                    document.getElementById('userinfo-result').innerHTML = 
                        '<div class="error"><strong>✗ UserInfo Error:</strong> ' + JSON.stringify(result) + '</div>';
                }
            } catch (error) {
                document.getElementById('userinfo-result').innerHTML = 
                    '<div class="error"><strong>✗ Error:</strong> ' + error.message + '</div>';
            }
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
