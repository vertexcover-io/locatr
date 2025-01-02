package locatr

import (
	"github.com/go-resty/resty/v2"
	"time"
)

func CreateNewHttpClient(baseUrl string) *resty.Client {
	client := resty.New()
	client.SetBaseURL(baseUrl)
	client.SetHeader("Accept", "application/json")
	client.SetTimeout(2 * time.Second)
	return client
}
