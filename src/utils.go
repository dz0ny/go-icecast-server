package utils

import (
	"bytes"
	"encoding/base64"
	"log"
	"strconv"
)

func CheckError(err error) {
	if err != nil {
		log.Println("Fatal error: ", err.Error())
	}
}

//parse data stream and return data until controll char
func Clean(tag []byte) string {
	start := bytes.Index(tag, []byte("="))
	end := len(tag)
	for i := 0; i < end; i++ {
		if tag[i] < 32 {
			end = i
			//lookahead
			if tag[i+1] < 32 {
				break
			}
		}
	}
	return Stringify(tag[start+1 : end])
}

// convert bytes to string
func Stringify(tag []byte) string {
	data := bytes.NewBuffer(tag[0:])

	return data.String()
}

//parse cli port to all interfaces port used by http package
func ToIfPort(port int) string {
	service := strconv.AppendInt([]byte(":"), int64(port), 10)
	return Stringify(service)
}

//create base64 encoded string from password and user
func Basic_auth(user string, pass string) string {
	encoded := &bytes.Buffer{}
	encoder := base64.NewEncoder(base64.StdEncoding, encoded)
	encoder.Write([]byte(user + ":" + pass))
	return encoded.String()
}
