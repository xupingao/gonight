package main

import (
	"github.com/xupingao/gonight"
)

func main() {
	engine := gonight.Default()
	engine.Run(":8080")
}
