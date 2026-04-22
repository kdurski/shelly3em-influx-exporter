package main

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"time"
)

func parseTime(value string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04", value)
}

func parseFloat(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

func randomString(len int) string {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}
