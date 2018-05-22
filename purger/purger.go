package purger

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var AMP_JSON_URL = "https://cdn.ampproject.org/caches.json"
var PRIVATEKEY *rsa.PrivateKey
var PUBLICKEY *rsa.PublicKey

type AmpCache struct {
	Id                         string
	Name                       string
	Docs                       string
	UpdateCacheApiDomainSuffix string
}

type AmpCacheJson struct {
	Caches []AmpCache
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	PRIVATEKEY_FILE := os.Getenv("PRIVATEKEY_FILE")
	PUBLICKEY_FILE := os.Getenv("PUBLICKEY_FILE")
	PRIVATEKEY = loadPrivateKey(PRIVATEKEY_FILE)
	PUBLICKEY = loadPublicKey(PUBLICKEY_FILE)
}

func loadPrivateKey(path string) *rsa.PrivateKey {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	block, _ := pem.Decode([]byte(content))
	key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
	return key
}

func loadPublicKey(path string) *rsa.PublicKey {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	block, _ := pem.Decode([]byte(content))
	pub, _ := x509.ParsePKIXPublicKey(block.Bytes)
	return pub.(*rsa.PublicKey)
}

func httpGet(url string) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status: %d\n", resp.StatusCode)
}

func getAmpCachesJson() AmpCacheJson {
	res, err := http.Get(AMP_JSON_URL)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var amp_cache_json AmpCacheJson
	jsonErr := json.Unmarshal([]byte(body), &amp_cache_json)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	return amp_cache_json
}

func createSignature(url string) []byte {
	message := []byte(url)
	hashed := sha256.Sum256(message)

	signature, err := rsa.SignPKCS1v15(rand.Reader, PRIVATEKEY, crypto.SHA256, hashed[:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = rsa.VerifyPKCS1v15(PUBLICKEY, crypto.SHA256, hashed[:], signature)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return signature
}

func getAmpCacheUrl(originUrl string, ac AmpCache) string {
	u, err := url.Parse(originUrl)
	if err != nil {
		log.Fatal(err)
	}
	originHostname := u.Hostname()
	hostname := originHostname
	hostname = strings.Replace(hostname, "-", "--", -1)
	hostname = strings.Replace(hostname, ".", "-", -1)
	u.Host = hostname + "." + ac.UpdateCacheApiDomainSuffix
	u.Path = "/c/s/" + originHostname + u.Path
	return u.String()
}

func getAmpRefreshUrl(cacheUrl string) string {
	u, err := url.Parse(cacheUrl)
	if err != nil {
		log.Fatal(err)
	}
	amp_ts := strconv.Itoa(int(time.Now().Unix()))
	pathBase := "/update-cache"

	q := u.Query()
	q.Set("amp_action", "flush")
	q.Set("amp_ts", amp_ts)

	u.RawQuery = q.Encode()

	urlPathWithQuery := pathBase + u.EscapedPath() + "?" + u.RawQuery
	binaryURLSignature := createSignature(urlPathWithQuery)
	urlSignature := base64.StdEncoding.EncodeToString(binaryURLSignature)

	urlSignature = strings.Replace(urlSignature, "=", "", -1)
	urlSignature = strings.Replace(urlSignature, "+", "-", -1)
	urlSignature = strings.Replace(urlSignature, "/", "_", -1)

	q.Set("amp_url_signature", urlSignature)
	u.RawQuery = q.Encode()
	u.Path = pathBase + u.Path

	return u.String()
}

func Purge(url string) {
	fmt.Printf("URL: %s\n", url)
	amp_cache_json := getAmpCachesJson()
	for _, amp_cache := range amp_cache_json.Caches {
		cache_url := getAmpCacheUrl(url, amp_cache)
		refresh_url := getAmpRefreshUrl(cache_url)
		httpGet(refresh_url)
	}
}
