package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func Marshal(m map[string]interface{}) string {
	if byt, err := json.Marshal(m); err != nil {
		Errorf(err.Error())
		return ""
	} else {
		return string(byt)
	}
}

func Unmarshal(str string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		Errorf(err.Error())
		return nil, err
	} else {
		return data, nil
	}
}

func HttpGet(path string, query map[string]string) {

	params := url.Values{}
	Url, _ := url.Parse(baseUrl + path)
	for index, value := range query {
		params.Set(index, value)
	}

	Url.RawQuery = params.Encode()
	urlPath := Url.String()
	fmt.Println(urlPath)
	response, err := http.Get(urlPath)
	if err != nil {
		println("ERR", err)
	}
	println(response)
}

func HttpPost(path string, body map[string]interface{}) {

}
