package main

func GetClient(url string) (client *Client) {
	client = &Client{
		Url:      url,
		MaxRetry: 5,
	}
	return
}
