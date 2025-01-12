
# Another Markdown File

This file contains more mermaid to be parsed and also a different code block to show that the parser understands only to read mermaid.

```go
package main

import "fmt"

func main() {
    fmt.Println("hello world")
}
```

```mermaid
sequenceDiagram
    actor Alice
    participant Bob
    Alice ->> Bob : hello()
    Bob -->> Alice : hey
```

```mermaid
sequenceDiagram
actor User
participant Server
participant Database

    User ->> Server: login("username", "password")
    Server -->> User: authenticationToken
    Server ->> Database: query("SELECT * FROM users")
    activate Database
    Database -->> Server: userData
    deactivate Database
```

```mermaid
sequenceDiagram
    actor Client
    participant Gateway
    participant AuthService
    participant UserService
    participant Logger

    Client ->> Gateway: request("login")
    Gateway ->> AuthService: validateCredentials("username", "password")
    loop Retry Authentication
        AuthService ->> UserService: fetchUser("username")
        UserService -->> AuthService: userDetails
    end
    AuthService -->> Gateway: authToken
    Gateway -->> Client: token

    activate Logger
    Logger ->> Gateway: logRequest("requestId", "status")
    deactivate Logger

    Client ->> Gateway: fetchData(token)
    Gateway ->> UserService: getUserData(authToken)
    activate UserService
    UserService -->> Gateway: userData
    deactivate UserService
    Gateway -->> Client: data

```