package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"strconv"
	"time"
)

func mustParseTime(string string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", string)
	if err != nil {
		log.Panic(err)
	}
	return t
}

func mustParseFloat(string string) float64 {
	f, err := strconv.ParseFloat(string, 64)
	if err != nil {
		log.Panic(err)
	}
	return f
}

func randomString(len int) string {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		log.Panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}
