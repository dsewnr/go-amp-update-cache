package main

import "github.com/dsewnr/go-amp-update-cache/purger"

func main() {
	url := "YOUR_AMP_URL"
	purger.Purge(url)
}
