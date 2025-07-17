package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	pdf "github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

func convertHTMLToPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid or empty HTML", http.StatusBadRequest)
		return
	}

	token := os.Getenv("TOKEN")

	authorization := r.Header.Get("Authorization")

	if authorization != "Bearer "+token {
		http.Error(w, "Invalid Content-Type, must be text/html or application/xhtml+xml", http.StatusBadRequest)
	}

	pdfg, err := pdf.NewPDFGenerator()
	if err != nil {
		http.Error(w, "Failed to create PDF generator", http.StatusInternalServerError)
		return
	}

	page := pdf.NewPageReader(bytes.NewReader(body))
	page.JavaScriptDelay.Set(3000)
	page.EnableLocalFileAccess.Set(true)

	pdfg.AddPage(page)
	pdfg.MarginTop.Set(10)
	pdfg.MarginBottom.Set(10)

	if err := pdfg.Create(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate PDF: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=output.pdf")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfg.Bytes())
}

func main() {
	http.HandleFunc("/generate-pdf", convertHTMLToPDF)

	fmt.Println("Servidor iniciado em http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
