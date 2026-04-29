package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/ledongthuc/pdf"
)

func main() {
	pdf.DebugOn = true
	content, err := readPdf("Csf_ALI130805KG5_08.04.2026.pdf")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(content)
}

func readPdf(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	buf.ReadFrom(b)
	return buf.String(), nil
}
