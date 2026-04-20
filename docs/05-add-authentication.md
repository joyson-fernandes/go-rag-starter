## Add authentication

The starter is anonymous by default — anyone who can reach `/api/query` can chat with the bot. For a public help widget that's usually fine. For an internal tool or a gated help center, you'll want to add auth.

## Three patterns

### 1. `X-User-ID` header (simplest)

The bot already honours an optional `X-User-ID` header. If set, conversations are associated with that ID; if absent, the conversation is anonymous. Nothing is enforced.

Add enforcement at your **ingress** (Traefik, Nginx, Cloudflare Access, Envoy) rather than in the bot itself. The ingress terminates the user's session, validates their identity, and rewrites requests to include `X-User-ID: <their-id>`. The bot trusts anything it receives because it's behind the authed perimeter.

A common pattern: your ingress (Traefik, Nginx, Cloudflare Access) has a JWT or session validator that sets `X-User-ID` as a trusted header on every request that reaches the bot.

Pros: no changes to the bot code. Works with any identity provider.
Cons: requires an ingress layer; local-dev bypasses it.

### 2. API-key header

Add a middleware that rejects requests without a matching API key. Edit `main.go`:

```go
func withAPIKey(next http.Handler) http.Handler {
    expected := os.Getenv("API_KEY")
    if expected == "" {
        return next // auth disabled
    }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/healthz" || r.URL.Path == "/" || r.URL.Path == "/widget.js" {
            next.ServeHTTP(w, r) // public endpoints
            return
        }
        if r.Header.Get("X-API-Key") != expected {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

Wrap your mux: `handler := withAPIKey(withLogging(mux))`.

Widget users send the key via `data-api-key`:

```html
<script src="https://bot.example.com/widget.js" data-api-key="sk-..."></script>
```

(You'd add a matching `headers['X-API-Key'] = apiKey` in `widget.js` `fetch` call.)

Pros: simple, no external dependencies.
Cons: the API key is visible to anyone who inspects the page. Only use for internal tools or server-to-server.

### 3. JWT at the bot (stateless)

Validate a Bearer token on every request. Use `github.com/golang-jwt/jwt/v5`:

```go
func withJWT(next http.Handler) http.Handler {
    secret := []byte(os.Getenv("JWT_SECRET"))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/healthz" || r.URL.Path == "/" || r.URL.Path == "/widget.js" {
            next.ServeHTTP(w, r)
            return
        }
        auth := r.Header.Get("Authorization")
        token := strings.TrimPrefix(auth, "Bearer ")
        if token == auth { // no "Bearer " prefix
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
            return secret, nil
        })
        if err != nil || !parsed.Valid {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }
        claims := parsed.Claims.(jwt.MapClaims)
        if uid, ok := claims["sub"].(string); ok {
            r.Header.Set("X-User-ID", uid)
        }
        next.ServeHTTP(w, r)
    })
}
```

Your auth service issues JWTs; your frontend attaches them to the widget's fetch calls (edit `widget.js` to read a token from `localStorage`).

Pros: stateless, scales horizontally, works across domains.
Cons: token rotation and logout require extra wiring.

## Rate limiting

Orthogonal to auth but usually wanted together. A simple in-process limiter:

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(rate.Limit(10), 20) // 10 req/s, burst 20
```

Wrap the mux with a limiter middleware. For per-IP limits, use `github.com/ulule/limiter/v3`.

In production behind Cloudflare or Traefik, rate-limit at the edge instead — cheaper and DDoS-resistant.

## Recommendation

- **Public help widget on a marketing site** → no auth at the bot, rate-limit at the edge.
- **Internal team tool** → API key or your SSO (via ingress `X-User-ID`).
- **Logged-in users of your SaaS** → JWT issued by your auth service, validated at the bot (or at an ingress that strips to `X-User-ID`).
