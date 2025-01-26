package utilhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// URLEncode encodes a string for safe inclusion in a URL query.
func URLEncode(input string) string {
	return url.QueryEscape(input)
}

type Resp struct {
	StatusCode int
	Body       []byte
}

// JoinURL encodes a string for safe inclusion in a URL query.
func JoinURL(baseURL string, queryParams map[string]string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %v", err)
	}

	if len(queryParams) != 0 {

		// Add query parameters from the map
		query := url.Values{}
		for key, value := range queryParams {
			query.Add(key, value)
		}
		parsedURL.RawQuery = query.Encode()
	}

	return parsedURL.String(), nil
}

func PostJSON(baseURL string, queryParams map[string]string,
	headers map[string]string, bodyJSON any,
) ([]byte, error) {
	// The URL to send the POST request to
	url, err := JoinURL(baseURL, queryParams)

	if err != nil {
		return nil, err
	}

	var data io.Reader

	if bodyJSON != nil {

		// Convert request data to JSON
		jsonData, err := json.Marshal(bodyJSON)
		if err != nil {

			return nil, fmt.Errorf("error marshalling JSON: %v", err)
		}
		data = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set(`Content-Type`, "application/json")
	// Authorization
	if len(headers) != 0 {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {

		return nil, fmt.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("error on http resp check: %v", resp.StatusCode)
	}

	return body, err
}

func PostFormURL(baseURL string, queryParams map[string]string,
	headers map[string]string, bodyForm url.Values,
) ([]byte, error) {
	// The URL to send the POST request to
	url, err := JoinURL(baseURL, queryParams)

	if err != nil {
		return nil, err
	}

	var data io.Reader

	if len(bodyForm) != 0 {
		// Convert request data to JSON
		data = strings.NewReader(bodyForm.Encode())
	}

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set(`Content-Type`, "application/x-www-form-urlencoded")
	// req.SetBasicAuth()
	if len(headers) != 0 {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {

		return nil, fmt.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("error on http resp check: %v", resp.StatusCode)
	}

	return body, err
}

func GetBytes(baseURL string, queryParams map[string]string,
	headers map[string]string,
) ([]byte, error) {
	// The URL to send the POST request to
	url, err := JoinURL(baseURL, queryParams)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Authorization
	if len(headers) != 0 {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {

		return nil, fmt.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("error on http resp check: %v", resp.StatusCode)
	}

	return body, err
}

// AppendURL join like path?args[0]=args[1]&args[2]=args[3]#args[4]
// ignore query key-value pair or fragment if is empty
func AppendURL(path string, args ...string) string {

	count := len(args)
	pairs := count / 2

	if pairs > 0 {

		u, err := url.Parse(path)
		_ = err // log

		query := u.Query()
		for i := 0; i < pairs; i++ {
			k := args[i*2]
			v := args[i*2+1]
			if k != "" && v != "" {
				query.Add(k, v) // this not keep order, internal sort by key on encode
			}

		}

		u.RawQuery = query.Encode()

		if count%2 == 1 {
			v := args[count-1]
			if v != "" {
				u.Fragment = args[count-1]
			}
		}

		return u.String()

	}

	return path
}
