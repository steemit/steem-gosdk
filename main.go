package main

import (
	"github.com/steemit/steemgosdk/client"
)

func GetClient(url string) (c *client.Client) {
	c = &client.Client{
		Url:      url,
		MaxRetry: 5,
	}
	return
}
