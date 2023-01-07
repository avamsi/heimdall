package main

import (
	_ "embed"

	"github.com/avamsi/clifr"
)

//go:generate clifr docs --out=clifr.docs
//go:embed clifr.docs
var docs []byte

func main() {
	clifr.Execute(docs, Heimdall{}, Bifrost{})
}
