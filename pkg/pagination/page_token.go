package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

func DecodePageToken(pageToken string) (int64, error) {
	bytes, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return 0, errors.New("page token is invalid")
	}
	token, err := strconv.ParseInt(string(bytes), 10, 64)
	if err != nil {
		return 0, errors.New("page token is invalid")
	}
	return token, nil
}

func EncodePageToken(n int64) (string, error) {
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v", n)))
	return token, nil
}
