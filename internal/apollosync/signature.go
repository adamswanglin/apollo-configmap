/*
 Copyright 2025 adamswanglin

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package apollosync

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

const (
	AUTHORIZATION_FORMAT      = "Apollo %s:%s"
	DELIMITER                 = "\n"
	HTTP_HEADER_AUTHORIZATION = "Authorization"
	HTTP_HEADER_TIMESTAMP     = "Timestamp"
)

// hmacSha1Sign computes the HMAC-SHA1 signature for the given string and secret.
func hmacSha1Sign(data, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// signature generates a signature using the given timestamp, pathWithQuery, and secret.
func signature(timestamp, pathWithQuery, secret string) string {
	stringToSign := timestamp + DELIMITER + pathWithQuery
	return hmacSha1Sign(stringToSign, secret)
}

// buildAuthorizationHeaders constructs the HTTP headers required for Apollo authorization.
func buildAuthorizationHeaders(urlStr, appId, secret string) (map[string]string, error) {
	currentTimeMillis := time.Now().UnixMilli()
	timestamp := fmt.Sprintf("%d", currentTimeMillis)

	pathWithQuery, err := urlToPathWithQuery(urlStr)
	if err != nil {
		return nil, err
	}

	signature := signature(timestamp, pathWithQuery, secret)

	headers := map[string]string{
		HTTP_HEADER_AUTHORIZATION: fmt.Sprintf(AUTHORIZATION_FORMAT, appId, signature),
		HTTP_HEADER_TIMESTAMP:     timestamp,
	}

	return headers, nil
}

// urlToPathWithQuery extracts the path and query from a URL.
func urlToPathWithQuery(urlStr string) (string, error) {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid url pattern: %s, %w", urlStr, err)
	}

	pathWithQuery := parsedUrl.Path
	if parsedUrl.RawQuery != "" {
		pathWithQuery += "?" + parsedUrl.RawQuery
	}

	return pathWithQuery, nil
}
