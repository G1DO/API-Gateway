package proxy

import (
    "io"
    "net/http"
    "time"
    "net"
    "context"
)

type proxy struct {
	url string
	client  *http.Client
}

func NewProxy(url string) *proxy {
    return &proxy{
        url: url,
        client: &http.Client{
            
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 100,
                IdleConnTimeout:     90 * time.Second,
                DialContext: (&net.Dialer{
    Timeout: 5 * time.Second,
}).DialContext,
            },
        },
    }
}

    
func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Build the backend URL: p.url + r.URL.Path
    //    use: backendURL := p.url + r.URL.Path
	backendURL := p.url + r.URL.Path
    // Right after line 36 (backendURL), add:
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()

    // 2. Create new request: http.NewRequest(method, url, body)
    //    method = r.Method
    //    url    = backendURL
    //    body   = r.Body
    
    newReq, err := http.NewRequestWithContext(ctx, r.Method, backendURL, r.Body)
	if err != nil{
		http.Error(w, "failed to create request", http.StatusInternalServerError)
    	return
	}


    // 3. Copy headers from r to your new request
    //    loop over r.Header and set them on your new request
    //    skip hop-by-hop headers
	hopByHop := map[string]bool{
    "Connection":          true,
    "Keep-Alive":          true,
    "Proxy-Authenticate":  true,
    "Proxy-Authorization": true,
    "Te":                  true,
    "Trailers":            true,
    "Transfer-Encoding":   true,
    "Upgrade":             true,
}

for key, values := range r.Header {
    if hopByHop[key] {
        continue
    }
    for _, v := range values {
        newReq.Header.Add(key, v)
    }
}
    // 4. Send the request: p.http.Do(newReq)
    //    this returns (resp, err)
     resp, err := p.client .Do(newReq)
    // 5. Handle error: if err != nil, write 502 to w
if err != nil {
    http.Error(w, "bad gateway", http.StatusBadGateway)
    return  // important! stop here
}
defer resp.Body.Close()

	for key, values := range resp.Header {
    for _, v := range values {
        w.Header().Add(key, v)
    }
}

    // 6. Copy response status: w.WriteHeader(resp.StatusCode)
	w.WriteHeader(resp.StatusCode)

    // 7. Copy response body: io.Copy(w, resp.Body)
	io.Copy(w, resp.Body)

}


