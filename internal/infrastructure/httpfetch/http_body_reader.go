package httpfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const maximumResponseBodyBytes = 10 << 20

type HttpBodyReader struct {
	httpClient *http.Client
}

func NewHttpBodyReader(httpClient *http.Client) *HttpBodyReader {
	return &HttpBodyReader{httpClient: httpClient}
}

func (httpBodyReader *HttpBodyReader) Read(ctx context.Context, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; go-symbol-news-hunter)")
	response, err := httpBodyReader.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status %d from %s", response.StatusCode, url)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(responseBody) > maximumResponseBodyBytes {
		return nil, fmt.Errorf("response from %s exceeds %d bytes", url, maximumResponseBodyBytes)
	}
	return responseBody, nil
}
