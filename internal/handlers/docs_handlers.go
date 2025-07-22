package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"oauth2-server/pkg/config"
)

// DocsHandler provides interactive API documentation
type DocsHandler struct {
	config *config.Config
}

// NewDocsHandler creates a new documentation handler
func NewDocsHandler(cfg *config.Config) *DocsHandler {
	return &DocsHandler{
		config: cfg,
	}
}

// ServeHTTP handles the documentation endpoint
func (h *DocsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/docs" {
		h.serveDocs(w, r)
		return
	}

	if r.URL.Path == "/docs/api.json" {
		h.serveOpenAPISpec(w, r)
		return
	}

	http.NotFound(w, r)
}

// serveDocs serves the interactive documentation UI
func (h *DocsHandler) serveDocs(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth2 Server API Documentation</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6; color: #333; background: #f8f9fa;
        }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { 
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white; padding: 40px 0; margin-bottom: 30px; border-radius: 8px;
        }
        .header h1 { font-size: 2.5rem; margin-bottom: 10px; }
        .header p { font-size: 1.1rem; opacity: 0.9; }
        
        .nav { 
            background: white; padding: 20px; border-radius: 8px; 
            margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .nav-links { display: flex; gap: 20px; flex-wrap: wrap; }
        .nav-link { 
            padding: 8px 16px; background: #007bff; color: white; 
            text-decoration: none; border-radius: 4px; transition: all 0.3s;
        }
        .nav-link:hover { background: #0056b3; transform: translateY(-1px); }
        
        .section { 
            background: white; margin-bottom: 30px; border-radius: 8px; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1); overflow: hidden;
        }
        .section-header { 
            background: #f8f9fa; padding: 20px; border-bottom: 1px solid #e9ecef;
        }
        .section-header h2 { color: #495057; display: flex; align-items: center; gap: 10px; }
        .section-content { padding: 20px; }
        
        .endpoint { 
            border: 1px solid #e9ecef; border-radius: 6px; 
            margin-bottom: 20px; overflow: hidden;
        }
        .endpoint-header { 
            display: flex; align-items: center; gap: 15px; 
            padding: 15px 20px; background: #f8f9fa; cursor: pointer;
            transition: background 0.3s;
        }
        .endpoint-header:hover { background: #e9ecef; }
        .method { 
            padding: 4px 12px; border-radius: 4px; font-weight: bold; 
            font-size: 0.85rem; text-transform: uppercase;
        }
        .method.get { background: #d4edda; color: #155724; }
        .method.post { background: #cce5ff; color: #004085; }
        .method.put { background: #fff3cd; color: #856404; }
        .method.delete { background: #f8d7da; color: #721c24; }
        
        .endpoint-details { 
            padding: 20px; background: white; display: none;
            border-top: 1px solid #e9ecef;
        }
        .endpoint-details.active { display: block; }
        
        .test-form { 
            background: #f8f9fa; padding: 20px; border-radius: 6px; 
            margin-top: 15px; border: 1px solid #e9ecef;
        }
        .form-group { margin-bottom: 15px; }
        .form-group label { display: block; margin-bottom: 5px; font-weight: 600; }
        .form-group input, .form-group select, .form-group textarea { 
            width: 100%%; padding: 8px 12px; border: 1px solid #ced4da; 
            border-radius: 4px; font-size: 14px;
        }
        .btn { 
            padding: 10px 20px; background: #007bff; color: white; 
            border: none; border-radius: 4px; cursor: pointer; 
            font-size: 14px; transition: all 0.3s;
        }
        .btn:hover { background: #0056b3; transform: translateY(-1px); }
        .btn-test { background: #28a745; }
        .btn-test:hover { background: #1e7e34; }
        
        .response { 
            margin-top: 15px; padding: 15px; background: #f8f9fa; 
            border-radius: 4px; border-left: 4px solid #007bff;
        }
        .response pre { 
            background: #343a40; color: #f8f9fa; padding: 15px; 
            border-radius: 4px; overflow-x: auto; font-size: 13px;
        }
        
        .status-indicator { 
            display: inline-block; width: 8px; height: 8px; 
            border-radius: 50%%; margin-right: 8px;
        }
        .status-online { background: #28a745; }
        .status-offline { background: #dc3545; }
        
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .card h3 { color: #495057; margin-bottom: 15px; }
        .card p { color: #6c757d; margin-bottom: 10px; }
        
        @media (max-width: 768px) {
            .container { padding: 10px; }
            .header h1 { font-size: 2rem; }
            .nav-links { flex-direction: column; }
            .endpoint-header { flex-direction: column; align-items: flex-start; gap: 10px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 OAuth2 Server API</h1>
            <p>Interactive API Documentation & Testing Interface</p>
            <p>Base URL: <code>%s</code></p>
        </div>

        <div class="nav">
            <div class="nav-links">
                <a href="#overview" class="nav-link">📋 Overview</a>
                <a href="#auth-endpoints" class="nav-link">🔐 Authentication</a>
                <a href="#token-endpoints" class="nav-link">🎫 Tokens</a>
                <a href="#device-flow" class="nav-link">📱 Device Flow</a>
                <a href="#client-mgmt" class="nav-link">👥 Client Management</a>
                <a href="#testing" class="nav-link">🧪 Testing</a>
            </div>
        </div>

        <div id="overview" class="section">
            <div class="section-header">
                <h2>📋 Server Overview</h2>
            </div>
            <div class="section-content">
                <div class="grid">
                    <div class="card">
                        <h3>🌐 Server Status</h3>
                        <p><span class="status-indicator status-online"></span>Online</p>
                        <p><strong>Base URL:</strong> %s</p>
                        <p><strong>Version:</strong> 1.0.0</p>
                    </div>
                    <div class="card">
                        <h3>🔧 Supported Flows</h3>
                        <p>✅ Authorization Code</p>
                        <p>✅ Client Credentials</p>
                        <p>✅ Device Flow</p>
                        <p>✅ Token Exchange</p>
                        <p>✅ Refresh Token</p>
                    </div>
                    <div class="card">
                        <h3>📊 Features</h3>
                        <p>✅ PKCE Support</p>
                        <p>✅ OpenID Connect</p>
                        <p>✅ Dynamic Registration</p>
                        <p>✅ Token Introspection</p>
                        <p>✅ Token Revocation</p>
                    </div>
                </div>
            </div>
        </div>

        <div id="auth-endpoints" class="section">
            <div class="section-header">
                <h2>🔐 Authentication Endpoints</h2>
            </div>
            <div class="section-content">
                %s
            </div>
        </div>

        <div id="token-endpoints" class="section">
            <div class="section-header">
                <h2>🎫 Token Endpoints</h2>
            </div>
            <div class="section-content">
                %s
            </div>
        </div>

        <div id="device-flow" class="section">
            <div class="section-header">
                <h2>📱 Device Flow</h2>
            </div>
            <div class="section-content">
                %s
            </div>
        </div>

        <div id="client-mgmt" class="section">
            <div class="section-header">
                <h2>👥 Client Management</h2>
            </div>
            <div class="section-content">
                %s
            </div>
        </div>

        <div id="testing" class="section">
            <div class="section-header">
                <h2>🧪 Quick Testing</h2>
            </div>
            <div class="section-content">
                <div class="card">
                    <h3>🚀 Quick Start</h3>
                    <p>Try the authorization code flow:</p>
                    <a href="/client1/auth" class="btn">Start Authorization Flow</a>
                </div>
            </div>
        </div>
    </div>

    <script>
        // Toggle endpoint details
        document.querySelectorAll('.endpoint-header').forEach(header => {
            header.addEventListener('click', () => {
                const details = header.nextElementSibling;
                details.classList.toggle('active');
            });
        });

        // Test endpoint functionality
        function testEndpoint(form, url, method = 'GET') {
            const formData = new FormData(form);
            const params = new URLSearchParams();
            
            for (let [key, value] of formData.entries()) {
                if (value) params.append(key, value);
            }

            const requestInit = {
                method: method,
                headers: {}
            };

            if (method === 'POST' || method === 'PUT') {
                requestInit.headers['Content-Type'] = 'application/x-www-form-urlencoded';
                requestInit.body = params.toString();
            } else if (method === 'GET' && params.toString()) {
                url += '?' + params.toString();
            }

            const responseDiv = form.parentElement.querySelector('.response') || 
                             form.parentElement.appendChild(document.createElement('div'));
            responseDiv.className = 'response';
            responseDiv.innerHTML = '<p>Making request...</p>';

            fetch(url, requestInit)
                .then(response => {
                    const statusColor = response.ok ? '#28a745' : '#dc3545';
                    return response.text().then(text => {
                        try {
                            const json = JSON.parse(text);
                            responseDiv.innerHTML = 
                                '<p><strong>Status:</strong> <span style="color:' + statusColor + '">' + 
                                response.status + ' ' + response.statusText + '</span></p>' +
                                '<pre>' + JSON.stringify(json, null, 2) + '</pre>';
                        } catch {
                            responseDiv.innerHTML = 
                                '<p><strong>Status:</strong> <span style="color:' + statusColor + '">' + 
                                response.status + ' ' + response.statusText + '</span></p>' +
                                '<pre>' + text + '</pre>';
                        }
                    });
                })
                .catch(error => {
                    responseDiv.innerHTML = '<p style="color:#dc3545"><strong>Error:</strong> ' + error.message + '</p>';
                });
        }

        // Add smooth scrolling for navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const target = document.querySelector(link.getAttribute('href'));
                if (target) {
                    target.scrollIntoView({ behavior: 'smooth' });
                }
            });
        });
    </script>
</body>
</html>`,
		h.config.BaseURL,
		h.config.BaseURL,
		h.generateAuthEndpoints(),
		h.generateTokenEndpoints(),
		h.generateDeviceFlowEndpoints(),
		h.generateClientMgmtEndpoints(),
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// generateAuthEndpoints creates HTML for authentication endpoints
func (h *DocsHandler) generateAuthEndpoints() string {
	return `
        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method get">GET</span>
                <span>/auth</span>
                <span>Authorization Endpoint</span>
            </div>
            <div class="endpoint-details">
                <p>Initiates the OAuth2 authorization code flow.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/auth', 'GET');">
                        <div class="form-group">
                            <label>Response Type:</label>
                            <select name="response_type">
                                <option value="code">code</option>
                                <option value="token">token</option>
                                <option value="id_token">id_token</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" value="frontend-app" placeholder="frontend-app">
                        </div>
                        <div class="form-group">
                            <label>Redirect URI:</label>
                            <input name="redirect_uri" value="` + h.config.BaseURL + `/client1/callback">
                        </div>
                        <div class="form-group">
                            <label>Scope:</label>
                            <input name="scope" value="openid profile email api:read">
                        </div>
                        <div class="form-group">
                            <label>State:</label>
                            <input name="state" value="xyz123">
                        </div>
                        <button type="submit" class="btn btn-test">Test Authorization</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method get">GET</span>
                <span>/userinfo</span>
                <span>User Info Endpoint</span>
            </div>
            <div class="endpoint-details">
                <p>Returns user information for the authenticated user.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/userinfo', 'GET');">
                        <div class="form-group">
                            <label>Authorization Header:</label>
                            <input name="authorization" placeholder="Bearer YOUR_ACCESS_TOKEN">
                        </div>
                        <button type="submit" class="btn btn-test">Test UserInfo</button>
                    </form>
                </div>
            </div>
        </div>
    `
}

// generateTokenEndpoints creates HTML for token endpoints
func (h *DocsHandler) generateTokenEndpoints() string {
	return `
        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method post">POST</span>
                <span>/token</span>
                <span>Token Endpoint</span>
            </div>
            <div class="endpoint-details">
                <p>Exchange authorization code for access token or handle other token grant types.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/token', 'POST');">
                        <div class="form-group">
                            <label>Grant Type:</label>
                            <select name="grant_type">
                                <option value="authorization_code">authorization_code</option>
                                <option value="client_credentials">client_credentials</option>
                                <option value="refresh_token">refresh_token</option>
                                <option value="urn:ietf:params:oauth:grant-type:device_code">device_code</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" value="frontend-app">
                        </div>
                        <div class="form-group">
                            <label>Client Secret:</label>
                            <input name="client_secret" value="frontend-secret">
                        </div>
                        <div class="form-group">
                            <label>Code (for authorization_code):</label>
                            <input name="code" placeholder="Authorization code from /auth">
                        </div>
                        <div class="form-group">
                            <label>Redirect URI (for authorization_code):</label>
                            <input name="redirect_uri" value="` + h.config.BaseURL + `/client1/callback">
                        </div>
                        <button type="submit" class="btn btn-test">Test Token Request</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method post">POST</span>
                <span>/introspect</span>
                <span>Token Introspection</span>
            </div>
            <div class="endpoint-details">
                <p>Get information about an access token.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/introspect', 'POST');">
                        <div class="form-group">
                            <label>Token:</label>
                            <input name="token" placeholder="Access token to introspect">
                        </div>
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" value="frontend-app">
                        </div>
                        <div class="form-group">
                            <label>Client Secret:</label>
                            <input name="client_secret" value="frontend-secret">
                        </div>
                        <button type="submit" class="btn btn-test">Test Introspection</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method post">POST</span>
                <span>/revoke</span>
                <span>Token Revocation</span>
            </div>
            <div class="endpoint-details">
                <p>Revoke an access or refresh token.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/revoke', 'POST');">
                        <div class="form-group">
                            <label>Token:</label>
                            <input name="token" placeholder="Token to revoke">
                        </div>
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" value="frontend-app">
                        </div>
                        <div class="form-group">
                            <label>Client Secret:</label>
                            <input name="client_secret" value="frontend-secret">
                        </div>
                        <button type="submit" class="btn btn-test">Test Revocation</button>
                    </form>
                </div>
            </div>
        </div>
    `
}

// generateDeviceFlowEndpoints creates HTML for device flow endpoints
func (h *DocsHandler) generateDeviceFlowEndpoints() string {
	return `
        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method post">POST</span>
                <span>/device_authorization</span>
                <span>Device Authorization</span>
            </div>
            <div class="endpoint-details">
                <p>Start the device authorization flow.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/device_authorization', 'POST');">
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" value="frontend-app">
                        </div>
                        <div class="form-group">
                            <label>Scope:</label>
                            <input name="scope" value="api:read api:write">
                        </div>
                        <button type="submit" class="btn btn-test">Test Device Authorization</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method get">GET</span>
                <span>/device</span>
                <span>Device Verification</span>
            </div>
            <div class="endpoint-details">
                <p>Device verification page for users to enter their code.</p>
                <div class="test-form">
                    <a href="/device" class="btn">Open Device Verification</a>
                </div>
            </div>
        </div>
    `
}

// generateClientMgmtEndpoints creates HTML for client management endpoints
func (h *DocsHandler) generateClientMgmtEndpoints() string {
	return `
        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method post">POST</span>
                <span>/register</span>
                <span>Dynamic Client Registration</span>
            </div>
            <div class="endpoint-details">
                <p>Register a new OAuth2 client dynamically.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/register', 'POST');">
                        <div class="form-group">
                            <label>Client Name:</label>
                            <input name="client_name" value="My Test Client">
                        </div>
                        <div class="form-group">
                            <label>Redirect URIs (JSON array):</label>
                            <textarea name="redirect_uris">["` + h.config.BaseURL + `/callback"]</textarea>
                        </div>
                        <div class="form-group">
                            <label>Grant Types (JSON array):</label>
                            <textarea name="grant_types">["authorization_code", "refresh_token"]</textarea>
                        </div>
                        <div class="form-group">
                            <label>Response Types (JSON array):</label>
                            <textarea name="response_types">["code"]</textarea>
                        </div>
                        <button type="submit" class="btn btn-test">Test Registration</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method get">GET</span>
                <span>/.well-known/oauth-authorization-server</span>
                <span>Server Metadata</span>
            </div>
            <div class="endpoint-details">
                <p>OAuth2 Authorization Server Metadata (RFC 8414).</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/.well-known/oauth-authorization-server', 'GET');">
                        <button type="submit" class="btn btn-test">Get Server Metadata</button>
                    </form>
                </div>
            </div>
        </div>
    `
}

// serveOpenAPISpec serves an OpenAPI specification
func (h *DocsHandler) serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "OAuth2 Authorization Server",
			"description": "A comprehensive OAuth2 and OpenID Connect server implementation",
			"version":     "1.0.0",
		},
		"servers": []map[string]interface{}{
			{"url": h.config.BaseURL, "description": "OAuth2 Server"},
		},
		"paths": map[string]interface{}{
			"/auth": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Authorization Endpoint",
					"description": "OAuth2 authorization endpoint for initiating authorization code flow",
					"parameters": []map[string]interface{}{
						{
							"name":        "response_type",
							"in":          "query",
							"required":    true,
							"description": "Response type (code, token, id_token)",
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "client_id",
							"in":          "query",
							"required":    true,
							"description": "Client identifier",
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "redirect_uri",
							"in":          "query",
							"required":    false,
							"description": "Redirect URI",
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "scope",
							"in":          "query",
							"required":    false,
							"description": "Requested scope",
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "state",
							"in":          "query",
							"required":    false,
							"description": "State parameter",
							"schema":      map[string]string{"type": "string"},
						},
					},
				},
			},
			"/token": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Token Endpoint",
					"description": "OAuth2 token endpoint for exchanging authorization codes for tokens",
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}
