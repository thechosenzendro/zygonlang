package main

import (
	"fmt"
	"os"
	"thechosenzendro/zygonlang/zygonlang"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		sourceCode, err := os.ReadFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		val, _ := zygonlang.Exec(string(sourceCode))
		if val != nil {
			fmt.Println(val.Inspect())
		} else {
			fmt.Println(val)
		}

	} else {
		fmt.Println("Zygon commands:")
		fmt.Println("	run <file_path> - runs a zygon file")
	}
}
