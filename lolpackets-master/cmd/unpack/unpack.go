package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
)

var inPath = flag.String("in", "", "input chunk or keyframe file")
var printSummary = flag.Bool("summary", false, "if true, print each block summary")
var printContents = flag.Bool("contents", false, "if true, print each block contents")

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	data, err := ioutil.ReadFile(*inPath)
	exitIf(err)

	exitIf(readBlocks(data))
}

func readBlocks(data []byte) (err error) {
	r := bytes.NewReader(data)

	for {
		b, err := ReadBlock(r)
		if err != nil {
			break
		}
		if *printSummary {
			b.PrintSummary()
		}
		if *printContents {
			b.PrintContents()
		}
	}

	return err
}

func exitIf(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
