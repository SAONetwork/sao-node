package main

import (
	"reflect"
	"sao-node/api"
)

func main() {

}

func GetAPIType(name, pkg string) (i interface{}) {
	switch pkg {
	case "api":
		switch name {
		case "GatewayApi ":
			i = &api.GatewayApiStruct{}
			t = reflect.TypeOf()
		}
	}
}
