<h1 align="center">NetFunnel-Go</h1>
<p align="center">A wrapper library for NetFunnel</p>

## Before use
> [!WARNING]
> NetFunnel API runs in a separate environment for each service you use, so you need to know in advance the API Endpoint of the NetFunnel for the service you want to use.

## Example
```go
const NETFUNNEL_API = "https://netfunnel.example.com/ts.wseq"
const APPLICATION_API = "https://api.example.com"

func main() {
  // 1. Create a Client struct for NetFunnel.
  nf := Netfunnel.Client{
    ApiEndpoint: NETFUNNEL_API,
    RetryInterval: 1 * time.Second,
  }

  // 2. Get a valid ticket for NetFunnel. If the queue is full, it will wait for a ticket to become valid.
  ticket := nf.GetTicket()

  // 3. Put the ticket into the cookie jar.
  jar, _ := cookiejar.New(nil)
  var cookies []*http.Cookie
  cookie := &http.Cookie{
    Name:  "NetFunnel_ID",
    Value: ticket.Id,
  }
  cookies = append(cookies, cookie)
  jar.SetCookies(APPLICATION_API, cookies)

  // 4. Now use it as you would normally use a cookie jar.
  client := http.Client{
    Jar: jar,
  }

  // 5. Done!
  client.Post(APPLICATION_API, "application/json", nil)
}

```
