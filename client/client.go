package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	libraryVersion = "v3"
	defaultBaseURL = "https://%s/"
	absolutePath   = "api/nutanix/" + libraryVersion
	userAgent      = "nutanix/" + libraryVersion
	mediaType      = "application/json"
)

//Client Config Configuration of the client
type Client struct {
	Credentials *Credentials

	// HTTP client used to communicate with the Nutanix API.
	client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client
	UserAgent string

	// Optional function called after every successful request made.
	onRequestCompleted RequestCompletionCallback
}

// RequestCompletionCallback defines the type of the request callback function
type RequestCompletionCallback func(*http.Request, *http.Response, interface{})

// Credentials needed username and password
type Credentials struct {
	URL      string
	Username string
	Password string
	Endpoint string
	Port     string
	Insecure bool
}

// NewClient returns a new Nutanix API client.
func NewClient(credentials *Credentials) (*Client, error) {

	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: credentials.Insecure}, // ignore expired SSL certificates
	}

	httpClient := http.DefaultClient

	httpClient.Transport = transCfg

	baseURL, err := url.Parse(fmt.Sprintf(defaultBaseURL, credentials.URL))

	if err != nil {
		return nil, err
	}

	c := &Client{credentials, httpClient, baseURL, userAgent, nil}

	return c, nil
}

// NewRequest creates a request
func (c *Client) NewRequest(ctx context.Context, method, urlStr string, body interface{}) (*http.Request, error) {
	rel, errp := url.Parse(absolutePath + urlStr)
	if errp != nil {
		return nil, errp
	}

	u := c.BaseURL.ResolveReference(rel)

	buf := new(bytes.Buffer)

	if body != nil {
		err := json.NewEncoder(buf).Encode(body)

		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", mediaType)
	req.Header.Add("Accept", mediaType)
	req.Header.Add("User-Agent", c.UserAgent)
	req.Header.Add("Authorization", "Basic "+
		base64.StdEncoding.EncodeToString([]byte(c.Credentials.Username+":"+c.Credentials.Password)))

	//utils.PrintToJSON(req, "REQUEST BODY")

	// requestDump, err := httputil.DumpRequestOut(req, true)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println("################")
	// fmt.Println("REQUEST")
	// fmt.Println(string(requestDump))

	return req, nil
}

// OnRequestCompleted sets the DO API request completion callback
func (c *Client) OnRequestCompleted(rc RequestCompletionCallback) {
	c.onRequestCompleted = rc
}

//Do performs request passed
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) error {

	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)

	if err != nil {
		return err
	}

	defer func() {
		if rerr := resp.Body.Close(); err == nil {
			err = rerr
		}
	}()

	err = CheckResponse(resp)

	if err != nil {
		return err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				return err
			}
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
			if err != nil {
				return err
			}
			// utils.PrintToJSON(v, "RESPONSE BODY")
		}
	}

	if c.onRequestCompleted != nil {
		c.onRequestCompleted(req, resp, v)
	}

	return err
}

//CheckResponse checks errors if exist errors in request
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	data, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return err
	}

	res := &ErrorResponse{}
	err = json.Unmarshal(data, res)
	if err != nil {
		return err
	}

	pretty, _ := json.MarshalIndent(res, "", "  ")
	return fmt.Errorf("Error: %s", string(pretty))
}

//ErrorResponse ...
type ErrorResponse struct {
	APIVersion  string            `json:"api_version"`
	Code        int64             `json:"code"`
	Kind        string            `json:"kind"`
	MessageList []MessageResource `json:"message_list"`
	State       string            `json:"state"`
}

//MessageResource ...
type MessageResource struct {

	// Custom key-value details relevant to the status.
	Details map[string]interface{} `json:"details,omitempty"`

	// If state is ERROR, a message describing the error.
	Message string `json:"message"`

	// If state is ERROR, a machine-readable snake-cased *string.
	Reason string `json:"reason"`
}

func (r *ErrorResponse) Error() string {
	err := ""
	for key, value := range r.MessageList {
		err = fmt.Sprintf("%d: {message:%s, reason:%s }", key, value.Message, value.Reason)
	}
	return err
}
