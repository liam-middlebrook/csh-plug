csh-auth
========

An @ComputerScienceHouse authentication wrapper for Gin.

## Usage

1. Create a CSHAuth Struct

```
csh := csh_auth.CSHAuth{}
```

2. Initialize your CSHAuth object

```
csh.Init(
    /* oidc_client_id */,       // The OIDC client ID
    /* oidc_client_secret */,   // The OIDC client Secret
    /* jwt_secret */,           // I just used a random sequence of > 16 characters
    /* state */,                // I just used a random sequence of > 16 characters
    /* server_host */,          // The domain your application will run from
    /* redirect_uri */,         // The OIDC redirect URI
    /* auth_uri */,             // The relative path for your authentication endpoint
)
```

3. Add required CSHAuth endpoints

```
r.GET("/auth/login", csh.AuthRequest) // This endpoint should match auth_uri
r.GET("/auth/redir", csh.AuthCallback) // This endpoint should match the relative portion of redirect_uri
r.Get("/auth/logout", csh.AuthLogout)
```

4. Add endpoints to be behind authentication

```
r.Get("/hidden/prize", csh.AuthWrapper(endpoint_hidden_prize))
```
