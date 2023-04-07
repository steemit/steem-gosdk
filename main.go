package main

import (
	"github.com/steemit/steemgosdk/internal"
)

func GetClient(url string) (client *internal.Client) {
	client = &internal.Client{
		Url:      url,
		MaxRetry: 5,
	}
	return
}
