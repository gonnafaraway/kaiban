package integration

import "encoding/base64"

func stdB64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
