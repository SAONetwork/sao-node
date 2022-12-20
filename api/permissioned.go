package api

import (
	"github.com/filecoin-project/go-jsonrpc/auth"
)

const (
	PermNone  auth.Permission = "none" // default
	PermRead  auth.Permission = "read"
	PermWrite auth.Permission = "write"
	PermAdmin auth.Permission = "admin"
)

var AllPermissions = []auth.Permission{PermNone, PermRead, PermWrite, PermAdmin}
var DefaultPerms = []auth.Permission{PermNone}

func permissionedProxies(in, out interface{}) {
	outs := GetInternalStructs(out)
	for _, o := range outs {
		auth.PermissionedProxy(AllPermissions, DefaultPerms, in, o)
	}
}

func PermissionedSaoNodeAPI(a SaoApi) SaoApi {
	var out SaoApiStruct
	permissionedProxies(a, &out)
	return &out
}
