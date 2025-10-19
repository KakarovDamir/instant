# Authentication Flow Examples

Practical examples demonstrating the complete passwordless authentication flow with code samples in multiple languages.

## Overview

The Instant Platform uses passwordless authentication with 6-digit email verification codes. This example walks through the complete flow from code request to authenticated API access.

## Flow Diagram

```
User                    Gateway               Auth Service          Redis           Database
  |                        |                        |                  |                 |
  |--Request Code--------->|                        |                  |                 |
  |    (email)             |--Forward Request------>|                  |                 |
  |                        |                        |--Generate Code-->|                 |
  |                        |                        |    (6 digits)    |                 |
  |                        |                        |--Store Code----->|                 |
  |                        |                        |   (10min TTL)    |                 |
  |                        |                        |--Log to Console->|                 |
  |<---200 OK--------------|-<--200 OK--------------|                  |                 |
  |                        |                        |                  |                 |
  |--Verify Code---------->|                        |                  |                 |
  |   (email + code)       |--Forward Request------>|                  |                 |
  |                        |                        |--Validate Code-->|                 |
  |                        |                        |                  |                 |
  |                        |                        |--Get/Create User----------------->|
  |                        |                        |                  |                 |
  |                        |                        |--Create Session->|                 |
  |                        |                        |   (1hr TTL)      |                 |
  |<---Session Cookie------|-<--Session Cookie------|                  |                 |
  |                        |                        |                  |                 |
  |--API Request---------->|                        |                  |                 |
  |   (with cookie)        |--Validate Session----->|----------------->|                 |
  |                        |   (middleware)         |                  |                 |
  |                        |--Forward to Service--->|                  |                 |
  |                        |   (+ X-User-ID header) |                  |                 |
  |<---API Response--------|-<--Response------------|                  |                 |
```

## Example 1: cURL (Bash)

Complete authentication flow using cURL:

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"
EMAIL="user@example.com"

echo "=== Passwordless Authentication Flow ==="

# Step 1: Request verification code
echo -e "\n1. Requesting verification code for $EMAIL..."
curl -X POST "$BASE_URL/auth/request-code" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\"}" \
  -s | jq

# Step 2: Get code from logs (in another terminal)
echo -e "\n2. Check logs for verification code:"
echo "   docker-compose logs -f auth-service | grep 'Verification code'"
echo "   Or: docker-compose logs auth-service | grep 'Verification code' | tail -1"

# Step 3: Prompt for code
read -p "Enter verification code: " CODE

# Step 4: Verify code and get session
echo -e "\n3. Verifying code and creating session..."
RESPONSE=$(curl -X POST "$BASE_URL/auth/verify-code" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"code\":\"$CODE\"}" \
  -c cookies.txt \
  -s)

echo "$RESPONSE" | jq

# Extract user info
USER_ID=$(echo "$RESPONSE" | jq -r '.user.id')
echo -e "\nAuthenticated as user: $USER_ID"

# Step 5: Access protected endpoint
echo -e "\n4. Accessing protected endpoint..."
curl "$BASE_URL/api/posts" \
  -b cookies.txt \
  -s | jq

# Step 6: Logout
echo -e "\n5. Logging out..."
curl -X POST "$BASE_URL/auth/logout" \
  -b cookies.txt \
  -c cookies.txt \
  -s | jq

# Step 7: Verify logout
echo -e "\n6. Verifying logout (should get 401)..."
curl "$BASE_URL/api/posts" \
  -b cookies.txt \
  -s -w "\nHTTP Status: %{http_code}\n"

echo -e "\n=== Authentication flow complete ==="
```

## Example 2: JavaScript (Node.js)

Using `node-fetch` for server-side authentication:

```javascript
const fetch = require('node-fetch');
const readline = require('readline');

const BASE_URL = 'http://localhost:8080';
const EMAIL = 'user@example.com';

// Cookie jar for session management
let sessionCookie = null;

async function requestCode(email) {
  console.log(`\n1. Requesting verification code for ${email}...`);

  const response = await fetch(`${BASE_URL}/auth/request-code`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email }),
  });

  const data = await response.json();
  console.log('Response:', data);

  console.log('\n⚠️  Check auth service logs for verification code:');
  console.log('   docker-compose logs auth-service | grep "Verification code"');
}

async function verifyCode(email, code) {
  console.log(`\n2. Verifying code ${code}...`);

  const response = await fetch(`${BASE_URL}/auth/verify-code`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, code }),
  });

  // Extract session cookie
  const cookies = response.headers.raw()['set-cookie'];
  if (cookies) {
    sessionCookie = cookies.find(c => c.startsWith('session_id='));
  }

  const data = await response.json();
  console.log('Response:', data);
  console.log(`\n✅ Authenticated as: ${data.user.email}`);

  return data.user;
}

async function accessProtectedEndpoint() {
  console.log('\n3. Accessing protected endpoint...');

  const response = await fetch(`${BASE_URL}/api/posts`, {
    headers: {
      'Cookie': sessionCookie,
    },
  });

  const data = await response.json();
  console.log('Posts:', data);
}

async function logout() {
  console.log('\n4. Logging out...');

  const response = await fetch(`${BASE_URL}/auth/logout`, {
    method: 'POST',
    headers: {
      'Cookie': sessionCookie,
    },
  });

  const data = await response.json();
  console.log('Response:', data);

  sessionCookie = null;
  console.log('✅ Logged out successfully');
}

async function verifyLogout() {
  console.log('\n5. Verifying logout (should fail)...');

  const response = await fetch(`${BASE_URL}/api/posts`, {
    headers: {
      'Cookie': sessionCookie || '',
    },
  });

  console.log(`HTTP Status: ${response.status}`);

  if (response.status === 401) {
    console.log('✅ Session correctly invalidated');
  }
}

// Prompt for code input
async function getCodeFromUser() {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    rl.question('Enter verification code: ', (code) => {
      rl.close();
      resolve(code.trim());
    });
  });
}

async function main() {
  try {
    console.log('=== Passwordless Authentication Flow ===');

    // Step 1: Request code
    await requestCode(EMAIL);

    // Step 2: Get code from user
    const code = await getCodeFromUser();

    // Step 3: Verify code
    await verifyCode(EMAIL, code);

    // Step 4: Access protected endpoint
    await accessProtectedEndpoint();

    // Step 5: Logout
    await logout();

    // Step 6: Verify logout
    await verifyLogout();

    console.log('\n=== Flow complete ===');
  } catch (error) {
    console.error('Error:', error.message);
  }
}

main();
```

## Example 3: JavaScript (Browser/React)

Using fetch API in a React application:

```javascript
// AuthService.js
class AuthService {
  constructor(baseURL = 'http://localhost:8080') {
    this.baseURL = baseURL;
  }

  async requestCode(email) {
    const response = await fetch(`${this.baseURL}/auth/request-code`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email }),
    });

    if (!response.ok) {
      throw new Error('Failed to request verification code');
    }

    return response.json();
  }

  async verifyCode(email, code) {
    const response = await fetch(`${this.baseURL}/auth/verify-code`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include', // Important: save cookies
      body: JSON.stringify({ email, code }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Verification failed');
    }

    return response.json();
  }

  async logout() {
    const response = await fetch(`${this.baseURL}/auth/logout`, {
      method: 'POST',
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Logout failed');
    }

    return response.json();
  }

  async fetchProtected(endpoint) {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      credentials: 'include', // Include session cookie
    });

    if (response.status === 401) {
      throw new Error('Unauthorized - please login');
    }

    if (!response.ok) {
      throw new Error('Request failed');
    }

    return response.json();
  }
}

export default new AuthService();

// LoginComponent.jsx
import React, { useState } from 'react';
import AuthService from './AuthService';

function LoginComponent() {
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [step, setStep] = useState('email'); // 'email' or 'code'
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleRequestCode = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      await AuthService.requestCode(email);
      setStep('code');
      alert('Verification code sent! Check your email.');
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyCode = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const result = await AuthService.verifyCode(email, code);
      console.log('Logged in as:', result.user);
      // Redirect or update app state
      window.location.href = '/dashboard';
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  if (step === 'email') {
    return (
      <form onSubmit={handleRequestCode}>
        <h2>Login</h2>
        {error && <p className="error">{error}</p>}
        <input
          type="email"
          placeholder="Enter your email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <button type="submit" disabled={loading}>
          {loading ? 'Sending...' : 'Send Code'}
        </button>
      </form>
    );
  }

  return (
    <form onSubmit={handleVerifyCode}>
      <h2>Verify Code</h2>
      <p>Code sent to: {email}</p>
      {error && <p className="error">{error}</p>}
      <input
        type="text"
        placeholder="Enter 6-digit code"
        value={code}
        onChange={(e) => setCode(e.target.value)}
        maxLength={6}
        required
      />
      <button type="submit" disabled={loading}>
        {loading ? 'Verifying...' : 'Verify'}
      </button>
      <button type="button" onClick={() => setStep('email')}>
        Change Email
      </button>
    </form>
  );
}

export default LoginComponent;
```

## Example 4: Go Client

Authentication client in Go:

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
)

const baseURL = "http://localhost:8080"

type AuthClient struct {
	client  *http.Client
	baseURL string
}

func NewAuthClient() *AuthClient {
	jar, _ := cookiejar.New(nil)
	return &AuthClient{
		client: &http.Client{
			Jar: jar,
		},
		baseURL: baseURL,
	}
}

func (c *AuthClient) RequestCode(email string) error {
	payload := map[string]string{"email": email}
	body, _ := json.Marshal(payload)

	resp, err := c.client.Post(
		c.baseURL+"/auth/request-code",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *AuthClient) VerifyCode(email, code string) (*User, error) {
	payload := map[string]string{
		"email": email,
		"code":  code,
	}
	body, _ := json.Marshal(payload)

	resp, err := c.client.Post(
		c.baseURL+"/auth/verify-code",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("verification failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Message string `json:"message"`
		User    User   `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.User, nil
}

func (c *AuthClient) FetchProtected(endpoint string) ([]byte, error) {
	resp, err := c.client.Get(c.baseURL + endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized - please login")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *AuthClient) Logout() error {
	resp, err := c.client.Post(c.baseURL+"/auth/logout", "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logout failed with status: %d", resp.StatusCode)
	}

	return nil
}

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func main() {
	client := NewAuthClient()
	email := "user@example.com"

	// Step 1: Request code
	fmt.Println("1. Requesting verification code...")
	if err := client.RequestCode(email); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Code sent! Check logs for code.")

	// Step 2: Get code from user
	var code string
	fmt.Print("Enter verification code: ")
	fmt.Scanln(&code)

	// Step 3: Verify code
	fmt.Println("\n2. Verifying code...")
	user, err := client.VerifyCode(email, code)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Authenticated as: %s (%s)\n", user.Email, user.ID)

	// Step 4: Access protected endpoint
	fmt.Println("\n3. Accessing protected endpoint...")
	data, err := client.FetchProtected("/api/posts")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response: %s\n", string(data))

	// Step 5: Logout
	fmt.Println("\n4. Logging out...")
	if err := client.Logout(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Logged out successfully")

	// Step 6: Verify logout
	fmt.Println("\n5. Verifying logout...")
	_, err = client.FetchProtected("/api/posts")
	if err != nil {
		fmt.Println("✅ Session correctly invalidated:", err)
	}
}
```

## Example 5: Python

Using `requests` library:

```python
import requests
import json

BASE_URL = "http://localhost:8080"

class AuthClient:
    def __init__(self):
        self.session = requests.Session()

    def request_code(self, email):
        """Request verification code"""
        response = self.session.post(
            f"{BASE_URL}/auth/request-code",
            json={"email": email}
        )
        response.raise_for_status()
        return response.json()

    def verify_code(self, email, code):
        """Verify code and create session"""
        response = self.session.post(
            f"{BASE_URL}/auth/verify-code",
            json={"email": email, "code": code}
        )
        response.raise_for_status()
        data = response.json()
        return data['user']

    def fetch_protected(self, endpoint):
        """Fetch protected endpoint"""
        response = self.session.get(f"{BASE_URL}{endpoint}")
        response.raise_for_status()
        return response.json()

    def logout(self):
        """Logout and clear session"""
        response = self.session.post(f"{BASE_URL}/auth/logout")
        response.raise_for_status()
        return response.json()

def main():
    client = AuthClient()
    email = "user@example.com"

    print("=== Passwordless Authentication Flow ===\n")

    # Step 1: Request code
    print(f"1. Requesting verification code for {email}...")
    result = client.request_code(email)
    print(f"   {result['message']}")
    print("\n⚠️  Check auth service logs for verification code")

    # Step 2: Get code from user
    code = input("\nEnter verification code: ").strip()

    # Step 3: Verify code
    print("\n2. Verifying code...")
    user = client.verify_code(email, code)
    print(f"   ✅ Authenticated as: {user['email']} ({user['id']})")

    # Step 4: Access protected endpoint
    print("\n3. Accessing protected endpoint...")
    posts = client.fetch_protected("/api/posts")
    print(f"   Received {len(posts)} posts")

    # Step 5: Logout
    print("\n4. Logging out...")
    result = client.logout()
    print(f"   ✅ {result['message']}")

    # Step 6: Verify logout
    print("\n5. Verifying logout (should fail)...")
    try:
        client.fetch_protected("/api/posts")
    except requests.HTTPError as e:
        if e.response.status_code == 401:
            print("   ✅ Session correctly invalidated")

    print("\n=== Flow complete ===")

if __name__ == "__main__":
    main()
```

## Testing Checklist

Use this checklist to verify authentication works correctly:

- [ ] Request code returns 200 OK
- [ ] Code appears in auth service logs
- [ ] Code expires after 10 minutes
- [ ] Invalid code returns 401 Unauthorized
- [ ] Expired code returns 401 Unauthorized
- [ ] Valid code returns session cookie
- [ ] Session cookie allows access to protected endpoints
- [ ] Missing cookie returns 401 Unauthorized
- [ ] Logout invalidates session
- [ ] Expired session returns 401 Unauthorized
- [ ] Multiple simultaneous sessions work independently

## Common Issues

### Code Not Appearing in Logs

**Solution:**
```bash
# Ensure auth service is running
docker-compose ps auth-service

# View logs in real-time
docker-compose logs -f auth-service | grep "Verification"
```

### Session Not Persisting

**Solution:**
- Ensure cookies are enabled (`credentials: 'include'` in fetch)
- Check `Set-Cookie` header in response
- Verify Redis is running: `docker-compose ps redis`

### 401 Unauthorized on Protected Endpoint

**Solution:**
- Verify session cookie is being sent
- Check session exists in Redis: `docker exec -it instant-redis-1 redis-cli KEYS session:*`
- Ensure gateway middleware is validating sessions

## Related Documentation

- [Authentication API](../api/auth.md) - Complete API reference
- [Session Package](../packages/session.md) - Session implementation details
- [Getting Started](../guides/getting-started.md) - Development setup
