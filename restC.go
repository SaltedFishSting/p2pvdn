// restC
package main

import (
	"errors"
	"net/http"
	"rest"
	"strconv"
	"time"
)

func init() {
	restClt.customClient = &rest.Client{&http.Client{Timeout: time.Millisecond * 3000}}
}

type RestClient struct {
	customClient *rest.Client
}

var restClt RestClient

func (rc *RestClient) post(api string) (string, error) {
	baseURL := globeCfg.Rest.Vdn + api

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json; charset=utf-8"

	method := rest.Post

	request := rest.Request{
		Method:      method,
		BaseURL:     baseURL,
		Headers:     headers,
		QueryParams: nil,
		Body:        nil,
	}

	response, err := rc.customClient.API(request)
	if err != nil {
		return "", errors.New("Post:" + err.Error())
	} else {
		if response.StatusCode != 200 {
			return "", errors.New("Confirm: http.respose.statuscode:" + strconv.FormatInt(int64(response.StatusCode), 10))
		}

		return response.Body, nil
	}
}

func QueryVdnMonitorData(api string) (string, error) {
	return restClt.post(api)
	//	rsp, err := restClt.post(api)
	//	if err != nil {
	//		return err
	//	}

	//	return parser(rsp, extractor)
}
