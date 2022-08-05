package utils

import (
	"errors"
	"log"
	"net"
	"net/http"
	"reflect"
)

func CopyOriginHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func AddCustomHeaders(dist http.Header, headers map[string]string) {
	for k, v := range headers {
		dist.Set(k, v)
	}
}

func Async(fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("recover from err: %v", err)
			}
		}()

		fn()
	}()
}

func GetIPAddr(req *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return "", err
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		return "", errors.New("Parsing IP from Request.RemoteAddr got nothing.")
	}

	return userIP.String(), nil
}

func ValidateConfig(config interface{}) bool {
	t := reflect.TypeOf(config).Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		val, ok := f.Tag.Lookup("required")
		if !ok {
			continue
		}

		isRequired := val == "true"

		switch f.Type.Kind() {
		case reflect.Array:
		case reflect.Slice:
		case reflect.Map:
			if isRequired && reflect.ValueOf(config).Elem().Field(i).IsNil() {
				return false
			}
		}
	}

	return true
}
