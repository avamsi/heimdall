package main

import (
	_ "embed"

	"github.com/avamsi/eclipse"
)

//go:generate eclipse docs --out=eclipse.docs
//go:embed eclipse.docs
var docs []byte

func main() {
	eclipse.Execute(docs, Heimdall{}, Bifrost{})
}
