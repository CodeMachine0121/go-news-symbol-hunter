package utilities

import (
	"fmt"
	"io"
	"net/http"
)

type HttpBodyReader struct {
	httpClient *http.Client
}

func NewHttpBodyReader(httpClient *http.Client) *HttpBodyReader {
	return &HttpBodyReader{httpClient: httpClient}
}

func (httpBodyReader *HttpBodyReader) Read(url string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
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
	return io.ReadAll(response.Body)
}
