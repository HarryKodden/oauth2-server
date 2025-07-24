package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"oauth2-server/internal/store"
	"oauth2-server/pkg/config"
)

// DocsHandler provides interactive API documentation
type DocsHandler struct {
	config      *config.Config
	clientStore *store.ClientStore
}

// NewDocsHandler creates a new documentation handler
func NewDocsHandler(cfg *config.Config, clientStore *store.ClientStore) *DocsHandler {
	return &DocsHandler{
		config:      cfg,
		clientStore: clientStore,
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

	// Handle client management API endpoints
	if r.URL.Path == "/docs/api/clients" {
		h.HandleClientsAPI(w, r)
		return
	}

	if len(r.URL.Path) > 18 && r.URL.Path[:18] == "/docs/api/clients/" {
		h.HandleClientAPI(w, r)
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
                <span class="method get">GET</span>
                <span>/api/clients</span>
                <span>List All Clients</span>
            </div>
            <div class="endpoint-details">
                <p>Retrieve a list of all registered OAuth2 clients.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); listClients();">
                        <button type="submit" class="btn btn-test">List All Clients</button>
                    </form>
                    <div id="client-response" class="response" style="display: none;"></div>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method post">POST</span>
                <span>/api/clients</span>
                <span>Create New Client</span>
            </div>
            <div class="endpoint-details">
                <p>Create a new OAuth2 client with specified parameters.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); createClient(this);">
                        <div class="form-group">
                            <label>Client Name:</label>
                            <input name="name" placeholder="My Application" required>
                        </div>
                        <div class="form-group">
                            <label>Description:</label>
                            <input name="description" placeholder="Application description">
                        </div>
                        <div class="form-group">
                            <label>Redirect URIs (comma-separated):</label>
                            <textarea name="redirect_uris" placeholder="/callback, /oauth/callback, https://app.example.com/callback"></textarea>
                        </div>
                        <div class="form-group">
                            <label>Grant Types (comma-separated):</label>
                            <input name="grant_types" value="authorization_code,refresh_token" placeholder="authorization_code,client_credentials">
                        </div>
                        <div class="form-group">
                            <label>Response Types (comma-separated):</label>
                            <input name="response_types" value="code" placeholder="code,token">
                        </div>
                        <div class="form-group">
                            <label>Scopes (comma-separated):</label>
                            <input name="scopes" value="openid,profile,email" placeholder="openid,profile,email">
                        </div>
                        <div class="form-group">
                            <label>
                                <input type="checkbox" name="public"> Public Client (no secret required)
                            </label>
                        </div>
                        <div class="form-group">
                            <label>Token Endpoint Auth Method:</label>
                            <select name="token_endpoint_auth_method">
                                <option value="client_secret_basic">client_secret_basic</option>
                                <option value="client_secret_post">client_secret_post</option>
                                <option value="none">none (for public clients)</option>
                            </select>
                        </div>
                        <button type="submit" class="btn btn-test">Create Client</button>
                    </form>
                    <div id="create-client-response" class="response" style="display: none;"></div>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method get">GET</span>
                <span>/api/clients/{client_id}</span>
                <span>Get Client Details</span>
            </div>
            <div class="endpoint-details">
                <p>Retrieve details for a specific client.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); getClient(this);">
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" placeholder="client_123" required>
                        </div>
                        <button type="submit" class="btn btn-test">Get Client</button>
                    </form>
                    <div id="get-client-response" class="response" style="display: none;"></div>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method put">PUT</span>
                <span>/api/clients/{client_id}</span>
                <span>Update Client</span>
            </div>
            <div class="endpoint-details">
                <p>Update an existing client's configuration.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); updateClient(this);">
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" placeholder="client_123" required>
                        </div>
                        <div class="form-group">
                            <label>Client Name:</label>
                            <input name="name" placeholder="Updated Application Name">
                        </div>
                        <div class="form-group">
                            <label>Description:</label>
                            <input name="description" placeholder="Updated description">
                        </div>
                        <div class="form-group">
                            <label>Redirect URIs (comma-separated):</label>
                            <textarea name="redirect_uris" placeholder="/callback, /oauth/callback"></textarea>
                        </div>
                        <div class="form-group">
                            <label>Grant Types (comma-separated):</label>
                            <input name="grant_types" placeholder="authorization_code,refresh_token">
                        </div>
                        <div class="form-group">
                            <label>Response Types (comma-separated):</label>
                            <input name="response_types" placeholder="code">
                        </div>
                        <div class="form-group">
                            <label>Scopes (comma-separated):</label>
                            <input name="scopes" placeholder="openid,profile,email">
                        </div>
                        <div class="form-group">
                            <label>
                                <input type="checkbox" name="public"> Public Client
                            </label>
                        </div>
                        <button type="submit" class="btn btn-test">Update Client</button>
                    </form>
                    <div id="update-client-response" class="response" style="display: none;"></div>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method delete">DELETE</span>
                <span>/api/clients/{client_id}</span>
                <span>Delete Client</span>
            </div>
            <div class="endpoint-details">
                <p>Delete a client permanently.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); deleteClient(this);">
                        <div class="form-group">
                            <label>Client ID:</label>
                            <input name="client_id" placeholder="client_123" required>
                        </div>
                        <button type="submit" class="btn btn-test" style="background: #dc3545;" onclick="return confirm('Are you sure you want to delete this client? This action cannot be undone.')">Delete Client</button>
                    </form>
                    <div id="delete-client-response" class="response" style="display: none;"></div>
                </div>
            </div>
        </div>

        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method post">POST</span>
                <span>/register</span>
                <span>Dynamic Client Registration (RFC 7591)</span>
            </div>
            <div class="endpoint-details">
                <p>Register a new OAuth2 client using the standard Dynamic Client Registration protocol.</p>
                <div class="test-form">
                    <form onsubmit="event.preventDefault(); testEndpoint(this, '/register', 'POST');">
                        <div class="form-group">
                            <label>Client Name:</label>
                            <input name="client_name" value="My Test Client">
                        </div>
                        <div class="form-group">
                            <label>Redirect URIs (JSON array):</label>
                            <textarea name="redirect_uris">["/callback", "/oauth/callback"]</textarea>
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

        <!-- Client Management Dashboard -->
        <div class="section" style="margin-top: 30px;">
            <div class="section-header">
                <h3>📋 Client Management Dashboard</h3>
            </div>
            <div class="section-content">
                <div style="margin-bottom: 20px;">
                    <button onclick="loadClientDashboard()" class="btn btn-test">Load Clients Dashboard</button>
                    <button onclick="refreshClientList()" class="btn" style="margin-left: 10px;">Refresh</button>
                </div>
                <div id="client-dashboard" style="display: none;">
                    <div id="client-list" class="grid"></div>
                </div>
            </div>
        </div>

        <script>
            // Client Management Functions
            function listClients() {
                showLoading('client-response');
                fetch('/api/clients')
                    .then(response => response.json())
                    .then(data => {
                        displayResponse('client-response', data, 'Client List Retrieved Successfully');
                    })
                    .catch(error => {
                        displayError('client-response', error.message);
                    });
            }

            function createClient(form) {
                const formData = new FormData(form);
                const clientData = {
                    name: formData.get('name'),
                    description: formData.get('description'),
                    redirect_uris: formData.get('redirect_uris') ? formData.get('redirect_uris').split(',').map(s => s.trim()) : [],
                    grant_types: formData.get('grant_types') ? formData.get('grant_types').split(',').map(s => s.trim()) : [],
                    response_types: formData.get('response_types') ? formData.get('response_types').split(',').map(s => s.trim()) : [],
                    scopes: formData.get('scopes') ? formData.get('scopes').split(',').map(s => s.trim()) : [],
                    public: formData.get('public') === 'on',
                    token_endpoint_auth_method: formData.get('token_endpoint_auth_method')
                };

                showLoading('create-client-response');
                fetch('/api/clients', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(clientData)
                })
                .then(response => response.json())
                .then(data => {
                    displayResponse('create-client-response', data, 'Client Created Successfully');
                    form.reset();
                })
                .catch(error => {
                    displayError('create-client-response', error.message);
                });
            }

            function getClient(form) {
                const formData = new FormData(form);
                const clientId = formData.get('client_id');
                
                showLoading('get-client-response');
                fetch('/api/clients/' + encodeURIComponent(clientId))
                    .then(response => {
                        if (response.status === 404) {
                            throw new Error('Client not found');
                        }
                        return response.json();
                    })
                    .then(data => {
                        displayResponse('get-client-response', data, 'Client Details Retrieved');
                    })
                    .catch(error => {
                        displayError('get-client-response', error.message);
                    });
            }

            function updateClient(form) {
                const formData = new FormData(form);
                const clientId = formData.get('client_id');
                
                const updateData = {};
                if (formData.get('name')) updateData.name = formData.get('name');
                if (formData.get('description')) updateData.description = formData.get('description');
                if (formData.get('redirect_uris')) updateData.redirect_uris = formData.get('redirect_uris').split(',').map(s => s.trim());
                if (formData.get('grant_types')) updateData.grant_types = formData.get('grant_types').split(',').map(s => s.trim());
                if (formData.get('response_types')) updateData.response_types = formData.get('response_types').split(',').map(s => s.trim());
                if (formData.get('scopes')) updateData.scopes = formData.get('scopes').split(',').map(s => s.trim());
                updateData.public = formData.get('public') === 'on';

                showLoading('update-client-response');
                fetch('/api/clients/' + encodeURIComponent(clientId), {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(updateData)
                })
                .then(response => {
                    if (response.status === 404) {
                        throw new Error('Client not found');
                    }
                    return response.json();
                })
                .then(data => {
                    displayResponse('update-client-response', data, 'Client Updated Successfully');
                })
                .catch(error => {
                    displayError('update-client-response', error.message);
                });
            }

            function deleteClient(form) {
                const formData = new FormData(form);
                const clientId = formData.get('client_id');
                
                showLoading('delete-client-response');
                fetch('/api/clients/' + encodeURIComponent(clientId), {
                    method: 'DELETE'
                })
                .then(response => {
                    if (response.status === 404) {
                        throw new Error('Client not found');
                    } else if (response.status === 204) {
                        displayResponse('delete-client-response', { message: 'Client deleted successfully' }, 'Client Deleted');
                    }
                })
                .catch(error => {
                    displayError('delete-client-response', error.message);
                });
            }

            function loadClientDashboard() {
                document.getElementById('client-dashboard').style.display = 'block';
                refreshClientList();
            }

            function refreshClientList() {
                const clientList = document.getElementById('client-list');
                clientList.innerHTML = '<p>Loading clients...</p>';
                
                fetch('/api/clients')
                    .then(response => response.json())
                    .then(clients => {
                        if (clients.length === 0) {
                            clientList.innerHTML = '<p>No clients found.</p>';
                            return;
                        }
                        
                        let html = '';
                        clients.forEach(client => {
                            html += createClientCard(client);
                        });
                        clientList.innerHTML = html;
                    })
                    .catch(error => {
                        clientList.innerHTML = '<p style="color: red;">Error loading clients: ' + error.message + '</p>';
                    });
            }

            function createClientCard(client) {
                const rawUris = (client.redirect_uris || []).join(', ');
                const resolvedUris = (client.redirect_uris_resolved || []).join(', ');
                
                return '<div class="card">' +
                    '<h4>' + (client.name || client.id) + '</h4>' +
                    '<p><strong>ID:</strong> ' + client.id + '</p>' +
                    '<p><strong>Description:</strong> ' + (client.description || 'No description') + '</p>' +
                    '<p><strong>Grant Types:</strong> ' + (client.grant_types || []).join(', ') + '</p>' +
                    '<p><strong>Scopes:</strong> ' + (client.scopes || []).join(', ') + '</p>' +
                    '<p><strong>Public:</strong> ' + (client.public ? 'Yes' : 'No') + '</p>' +
                    '<p><strong>Redirect URIs (config):</strong> ' + rawUris + '</p>' +
                    '<p><strong>Redirect URIs (resolved):</strong> <em>' + resolvedUris + '</em></p>' +
                    '<div style="margin-top: 15px;">' +
                    '<button onclick="editClient(\'' + client.id + '\')" class="btn" style="margin-right: 10px;">Edit</button>' +
                    '<button onclick="deleteClientFromCard(\'' + client.id + '\')" class="btn" style="background: #dc3545;">Delete</button>' +
                    '</div>' +
                    '</div>';
            }

            function editClient(clientId) {
                // Pre-fill the update form
                const updateForm = document.querySelector('form[onsubmit*="updateClient"]');
                updateForm.querySelector('input[name="client_id"]').value = clientId;
                
                // Scroll to the update form
                updateForm.scrollIntoView({ behavior: 'smooth' });
                
                // Load current client data to pre-fill other fields
                fetch('/api/clients/' + encodeURIComponent(clientId))
                    .then(response => response.json())
                    .then(client => {
                        updateForm.querySelector('input[name="name"]').value = client.name || '';
                        updateForm.querySelector('input[name="description"]').value = client.description || '';
                        updateForm.querySelector('textarea[name="redirect_uris"]').value = (client.redirect_uris || []).join(', ');
                        updateForm.querySelector('input[name="grant_types"]').value = (client.grant_types || []).join(', ');
                        updateForm.querySelector('input[name="response_types"]').value = (client.response_types || []).join(', ');
                        updateForm.querySelector('input[name="scopes"]').value = (client.scopes || []).join(', ');
                        updateForm.querySelector('input[name="public"]').checked = client.public || false;
                    })
                    .catch(error => {
                        console.error('Error loading client data:', error);
                    });
            }

            function deleteClientFromCard(clientId) {
                if (confirm('Are you sure you want to delete client "' + clientId + '"? This action cannot be undone.')) {
                    fetch('/api/clients/' + encodeURIComponent(clientId), {
                        method: 'DELETE'
                    })
                    .then(response => {
                        if (response.status === 204) {
                            alert('Client deleted successfully');
                            refreshClientList();
                        } else {
                            throw new Error('Failed to delete client');
                        }
                    })
                    .catch(error => {
                        alert('Error deleting client: ' + error.message);
                    });
                }
            }

            // Utility functions
            function showLoading(elementId) {
                const element = document.getElementById(elementId) || createResponseElement(elementId);
                element.innerHTML = '<p>Loading...</p>';
                element.style.display = 'block';
            }

            function displayResponse(elementId, data, title) {
                const element = document.getElementById(elementId) || createResponseElement(elementId);
                element.innerHTML = 
                    '<p><strong>' + title + '</strong></p>' +
                    '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
                element.style.display = 'block';
            }

            function displayError(elementId, message) {
                const element = document.getElementById(elementId) || createResponseElement(elementId);
                element.innerHTML = '<p style="color: red;"><strong>Error:</strong> ' + message + '</p>';
                element.style.display = 'block';
            }

            function createResponseElement(elementId) {
                const element = document.createElement('div');
                element.id = elementId;
                element.className = 'response';
                element.style.display = 'none';
                
                // Find the form and add the response element after it
                const forms = document.querySelectorAll('form');
                for (let form of forms) {
                    if (form.getAttribute('onsubmit') && form.getAttribute('onsubmit').includes(elementId.replace('-response', ''))) {
                        form.parentNode.appendChild(element);
                        break;
                    }
                }
                
                return element;
            }
        </script>
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

// HandleClientsAPI handles the clients list API endpoint
func (h *DocsHandler) HandleClientsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listClients(w, r)
	case "POST":
		h.createClient(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleClientAPI handles individual client API endpoints
func (h *DocsHandler) HandleClientAPI(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Path[13:] // Remove "/api/clients/" prefix

	switch r.Method {
	case "GET":
		h.getClient(w, r, clientID)
	case "PUT":
		h.updateClient(w, r, clientID)
	case "DELETE":
		h.deleteClient(w, r, clientID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listClients returns all registered clients
func (h *DocsHandler) listClients(w http.ResponseWriter, r *http.Request) {
	clients := h.clientStore.ListClients()

	var clientList []map[string]interface{}
	for _, client := range clients {
		if storeClient, ok := client.(*store.Client); ok {
			clientInfo := map[string]interface{}{
				"id":                         storeClient.GetID(),
				"name":                       storeClient.Name,
				"description":                storeClient.Description,
				"redirect_uris":              storeClient.GetRedirectURIs(),
				"grant_types":                storeClient.GetGrantTypes(),
				"response_types":             storeClient.GetResponseTypes(),
				"scopes":                     storeClient.GetScopes(),
				"audience":                   storeClient.GetAudience(),
				"token_endpoint_auth_method": storeClient.TokenEndpointAuthMethod,
				"public":                     storeClient.IsPublic(),
				"enabled_flows":              storeClient.EnabledFlows,
			}
			clientList = append(clientList, clientInfo)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clientList)
}

// getClient returns a specific client's details
func (h *DocsHandler) getClient(w http.ResponseWriter, r *http.Request, clientID string) {
	client, err := h.clientStore.GetClient(r.Context(), clientID)
	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	if storeClient, ok := client.(*store.Client); ok {
		clientInfo := map[string]interface{}{
			"id":                         storeClient.GetID(),
			"name":                       storeClient.Name,
			"description":                storeClient.Description,
			"redirect_uris":              storeClient.GetRedirectURIs(),
			"grant_types":                storeClient.GetGrantTypes(),
			"response_types":             storeClient.GetResponseTypes(),
			"scopes":                     storeClient.GetScopes(),
			"audience":                   storeClient.GetAudience(),
			"token_endpoint_auth_method": storeClient.TokenEndpointAuthMethod,
			"public":                     storeClient.IsPublic(),
			"enabled_flows":              storeClient.EnabledFlows,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clientInfo)
		return
	}

	http.Error(w, "Invalid client type", http.StatusInternalServerError)
}

// createClient creates a new client
func (h *DocsHandler) createClient(w http.ResponseWriter, r *http.Request) {
	var clientData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&clientData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	name, ok := clientData["name"].(string)
	if !ok || name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Generate client ID and secret
	clientID := fmt.Sprintf("client_%d", len(h.clientStore.ListClients())+1)
	clientSecret := fmt.Sprintf("secret_%d", len(h.clientStore.ListClients())+1)

	// Extract arrays safely
	var redirectURIs []string
	if uris, ok := clientData["redirect_uris"].([]interface{}); ok {
		for _, uri := range uris {
			if uriStr, ok := uri.(string); ok {
				redirectURIs = append(redirectURIs, uriStr)
			}
		}
	}

	var grantTypes []string
	if grants, ok := clientData["grant_types"].([]interface{}); ok {
		for _, grant := range grants {
			if grantStr, ok := grant.(string); ok {
				grantTypes = append(grantTypes, grantStr)
			}
		}
	}
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	var responseTypes []string
	if responses, ok := clientData["response_types"].([]interface{}); ok {
		for _, response := range responses {
			if responseStr, ok := response.(string); ok {
				responseTypes = append(responseTypes, responseStr)
			}
		}
	}
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	var scopes []string
	if clientScopes, ok := clientData["scopes"].([]interface{}); ok {
		for _, scope := range clientScopes {
			if scopeStr, ok := scope.(string); ok {
				scopes = append(scopes, scopeStr)
			}
		}
	}
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	// Create the new client
	newClient := &store.Client{
		ID:                      clientID,
		Secret:                  []byte(clientSecret),
		Name:                    name,
		Description:             getStringValue(clientData, "description"),
		RedirectURIs:            redirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		Scopes:                  scopes,
		Public:                  getBoolValue(clientData, "public"),
		TokenEndpointAuthMethod: getStringValue(clientData, "token_endpoint_auth_method"),
	}

	if newClient.TokenEndpointAuthMethod == "" {
		newClient.TokenEndpointAuthMethod = "client_secret_basic"
	}

	err := h.clientStore.StoreClient(newClient)
	if err != nil {
		http.Error(w, "Failed to store client", http.StatusInternalServerError)
		return
	}

	// Return the created client with credentials
	response := map[string]interface{}{
		"id":                         newClient.GetID(),
		"secret":                     string(newClient.Secret),
		"name":                       newClient.Name,
		"description":                newClient.Description,
		"redirect_uris":              newClient.GetRedirectURIs(),
		"grant_types":                newClient.GetGrantTypes(),
		"response_types":             newClient.GetResponseTypes(),
		"scopes":                     newClient.GetScopes(),
		"audience":                   newClient.GetAudience(),
		"token_endpoint_auth_method": newClient.TokenEndpointAuthMethod,
		"public":                     newClient.IsPublic(),
		"enabled_flows":              newClient.EnabledFlows,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// updateClient updates an existing client
func (h *DocsHandler) updateClient(w http.ResponseWriter, r *http.Request, clientID string) {
	// Check if client exists
	existingClient, err := h.clientStore.GetClient(r.Context(), clientID)
	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get the existing client as our base
	storeClient, ok := existingClient.(*store.Client)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update fields if provided
	if name, ok := updateData["name"].(string); ok && name != "" {
		storeClient.Name = name
	}

	if description, ok := updateData["description"].(string); ok {
		storeClient.Description = description
	}

	// Update arrays if provided
	if uris, ok := updateData["redirect_uris"].([]interface{}); ok {
		var redirectURIs []string
		for _, uri := range uris {
			if uriStr, ok := uri.(string); ok {
				redirectURIs = append(redirectURIs, uriStr)
			}
		}
		storeClient.RedirectURIs = redirectURIs
	}

	if grants, ok := updateData["grant_types"].([]interface{}); ok {
		var grantTypes []string
		for _, grant := range grants {
			if grantStr, ok := grant.(string); ok {
				grantTypes = append(grantTypes, grantStr)
			}
		}
		storeClient.GrantTypes = grantTypes
	}

	if responses, ok := updateData["response_types"].([]interface{}); ok {
		var responseTypes []string
		for _, response := range responses {
			if responseStr, ok := response.(string); ok {
				responseTypes = append(responseTypes, responseStr)
			}
		}
		storeClient.ResponseTypes = responseTypes
	}

	if clientScopes, ok := updateData["scopes"].([]interface{}); ok {
		var scopes []string
		for _, scope := range clientScopes {
			if scopeStr, ok := scope.(string); ok {
				scopes = append(scopes, scopeStr)
			}
		}
		storeClient.Scopes = scopes
	}

	if public, ok := updateData["public"].(bool); ok {
		storeClient.Public = public
	}

	if authMethod, ok := updateData["token_endpoint_auth_method"].(string); ok && authMethod != "" {
		storeClient.TokenEndpointAuthMethod = authMethod
	}

	// Store the updated client
	err = h.clientStore.StoreClient(storeClient)
	if err != nil {
		http.Error(w, "Failed to update client", http.StatusInternalServerError)
		return
	}

	// Return the updated client
	response := map[string]interface{}{
		"id":                         storeClient.GetID(),
		"name":                       storeClient.Name,
		"description":                storeClient.Description,
		"redirect_uris":              storeClient.GetRedirectURIs(),
		"grant_types":                storeClient.GetGrantTypes(),
		"response_types":             storeClient.GetResponseTypes(),
		"scopes":                     storeClient.GetScopes(),
		"audience":                   storeClient.GetAudience(),
		"token_endpoint_auth_method": storeClient.TokenEndpointAuthMethod,
		"public":                     storeClient.IsPublic(),
		"enabled_flows":              storeClient.EnabledFlows,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// deleteClient deletes a client
func (h *DocsHandler) deleteClient(w http.ResponseWriter, r *http.Request, clientID string) {
	err := h.clientStore.DeleteClient(clientID)
	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func getBoolValue(data map[string]interface{}, key string) bool {
	if value, ok := data[key].(bool); ok {
		return value
	}
	return false
}
