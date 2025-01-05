package main

import (
	"fmt"

	locatr "github.com/vertexcover-io/locatr/golang"
)

func main() {
	options := locatr.BaseLocatrOptions{}
	_, err := locatr.NewAppiumLocatr("http://172.30.192.1:4723/", "bc57285a-c329-459b-ac08-14509371cb31", options)
	fmt.Println(err)

}
