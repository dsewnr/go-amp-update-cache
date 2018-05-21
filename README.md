# AMP update-cache in Go

參考 [AMP update-cache Demo](https://github.com/ampproject/amp-publisher-sample/tree/master/amp-update-cache) 並用 Golang 實現，詳細設定請先參閱 [Update AMP Content](https://developers.google.com/amp/cache/update-cache#update-cache-request) 說明文件。

## 用法
1. 產生 .env 檔，並設定私鑰及公鑰路徑
```
PRIVATEKEY_FILE="PATH/TO/PRIVATEKEY"
PUBLICKEY_FILE="PATH/TO/PUBLICKEY"
```
2. 呼叫 Purge 函式並將 AMP URL 代入
```
package main

import "github.com/dsewnr/go-amp-update-cache/purger"

func main() {
        url := "YOUR_AMP_URL"
        purger.Purge(url)
}
```
