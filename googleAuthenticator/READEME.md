## Usage
```go
package main

import (
    "fmt"

    "github.com/jjonline/share-mod-lib/googleAuthenticator"
)

func main {
    // generate key
    formattedKey := googleauthenticator.GenerateKey()
    authenticator := googleauthenticator.NewAuthenticator("issuer", "xxx@gmail.com", formattedKey)
    // generate uri for show
    uri := authenticator.GenerateTotpUri()
    fmt.Println(uri)
    // verify token
    passcode := "<from input>"
    if authenticator.VerifyToken(passcode) {
        // ok
    }
}