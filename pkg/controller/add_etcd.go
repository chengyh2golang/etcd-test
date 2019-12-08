package controller

import (
	"etcd-test/pkg/controller/etcd"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, etcd.Add)
}
