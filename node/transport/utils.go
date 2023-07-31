package transport

import (
	"io/ioutil"
	"net/http"
)

func GetExternalIp() (string, error) {
	res, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
