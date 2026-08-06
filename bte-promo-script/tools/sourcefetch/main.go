package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	dest := flag.String("dest", "", "destination directory for extracted episode text files")
	flag.Parse()
	urls := flag.Args()
	if *dest == "" {
		fatal("dest is required")
	}
	if len(urls) == 0 {
		if raw := strings.TrimSpace(os.Getenv("PROMO_SOURCE_DOCUMENTS")); raw != "" {
			if err := json.Unmarshal([]byte(raw), &urls); err != nil {
				fatal("PROMO_SOURCE_DOCUMENTS: %v", err)
			}
		}
	}
	if len(urls) == 0 {
		fatal("at least one source document URL is required")
	}

	if err := os.MkdirAll(*dest, 0o755); err != nil {
		fatal("mkdir dest: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	for i, rawURL := range urls {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}
		name := fmt.Sprintf("ep%03d.txt", i+1)
		outPath := filepath.Join(*dest, name)
		if err := fetchAndExtract(client, rawURL, outPath); err != nil {
			fatal("source %d (%s): %v", i+1, rawURL, err)
		}
		if (i+1)%10 == 0 || i+1 == len(urls) {
			fmt.Fprintf(os.Stderr, "promo-source-fetch: progress %d/%d\n", i+1, len(urls))
		}
		fmt.Fprintf(os.Stderr, "promo-source-fetch: saved %s <= %s\n", outPath, rawURL)
	}
}

func fetchAndExtract(client *http.Client, url, outPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if isPlainTextURL(url) {
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
		if err != nil {
			return err
		}
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return fmt.Errorf("empty text response")
		}
		return os.WriteFile(outPath, []byte(text+"\n"), 0o644)
	}

	tmp, err := os.CreateTemp(filepath.Dir(outPath), "source-*.docx")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	text, err := docxText(tmpPath)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("no text extracted from docx")
	}
	return os.WriteFile(outPath, []byte(text+"\n"), 0o644)
}

func isPlainTextURL(rawURL string) bool {
	path := strings.ToLower(rawURL)
	return strings.HasSuffix(path, ".txt") || strings.HasSuffix(path, ".md")
}

func docxText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var doc *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", fmt.Errorf("word/document.xml not found")
	}

	rc, err := doc.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if el.Name.Local == "t" {
				var text string
				if err := dec.DecodeElement(&text, &el); err != nil {
					return "", err
				}
				b.WriteString(text)
			} else if el.Name.Local == "tab" {
				b.WriteByte('\t')
			} else if el.Name.Local == "br" || el.Name.Local == "cr" {
				b.WriteByte('\n')
			}
		case xml.EndElement:
			if el.Name.Local == "p" {
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "promo-source-fetch: "+format+"\n", args...)
	os.Exit(1)
}
