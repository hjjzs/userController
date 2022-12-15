package utils

import (
	"crypto/md5"
	"fmt"
)


func MD5(str string) string {
	data := []byte(str)
	b := md5.Sum(data)
	return fmt.Sprintf("%x", b)
}
