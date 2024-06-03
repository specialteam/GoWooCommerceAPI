package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "time"

    retryablehttp "github.com/hashicorp/go-retryablehttp"
)

type WooCommerceAPI struct {
    url        string
    auth       string
    httpClient *retryablehttp.Client
}

func NewWooCommerceAPI(url, consumerKey, consumerSecret string, timeout time.Duration, maxRetries int) *WooCommerceAPI {
    client := retryablehttp.NewClient()
    client.HTTPClient.Timeout = timeout
    client.RetryMax = maxRetries

    return &WooCommerceAPI{
        url:        url,
        auth:       fmt.Sprintf("%s:%s", consumerKey, consumerSecret),
        httpClient: client,
    }
}

func (api *WooCommerceAPI) request(method, endpoint string, data interface{}, params map[string]string) ([]byte, error) {
    url := fmt.Sprintf("%s/wp-json/wc/v3/%s", api.url, endpoint)
    var reqBody []byte
    var err error

    if data != nil {
        reqBody, err = json.Marshal(data)
        if err != nil {
            return nil, err
        }
    }

    req, err := retryablehttp.NewRequest(method, url, bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, err
    }

    req.SetBasicAuth(api.auth, "")
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

    q := req.URL.Query()
    for key, value := range params {
        q.Add(key, value)
    }
    req.URL.RawQuery = q.Encode()

    resp, err := api.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("API request failed: %v", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
    }

    return body, nil
}

func (api *WooCommerceAPI) Get(endpoint string, params map[string]string) ([]byte, error) {
    return api.request("GET", endpoint, nil, params)
}

func (api *WooCommerceAPI) Post(endpoint string, data interface{}) ([]byte, error) {
    return api.request("POST", endpoint, data, nil)
}

func (api *WooCommerceAPI) Put(endpoint string, data interface{}) ([]byte, error) {
    return api.request("PUT", endpoint, data, nil)
}

func (api *WooCommerceAPI) Delete(endpoint string) ([]byte, error) {
    return api.request("DELETE", endpoint, nil, nil)
}

func (api *WooCommerceAPI) GetAllProducts(perPage int) ([]map[string]interface{}, error) {
    var products []map[string]interface{}
    page := 1

    for {
        params := map[string]string{
            "per_page": fmt.Sprintf("%d", perPage),
            "page":     fmt.Sprintf("%d", page),
        }
        body, err := api.Get("products", params)
        if err != nil {
            return nil, err
        }

        var pageProducts []map[string]interface{}
        if err := json.Unmarshal(body, &pageProducts); err != nil {
            return nil, err
        }

        if len(pageProducts) == 0 {
            break
        }

        products = append(products, pageProducts...)
        page++
    }

    return products, nil
}

func main() {
    api := NewWooCommerceAPI("https://example.com", "consumer_key", "consumer_secret", 10*time.Second, 3)

    products, err := api.GetAllProducts(100)
    if err != nil {
        log.Fatalf("Failed to get products: %v", err)
    }

    for _, product := range products {
        fmt.Println(product)
    }
}
