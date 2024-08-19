package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

func main() {
	links, err := readLinksFromFile("new_links.txt")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("info: input links:")
	for _, link := range links {
		fmt.Println(link)
	}

	maxConcurrentDownloads := 5 // threads power
	downloadSem := make(chan struct{}, maxConcurrentDownloads)

	var wg sync.WaitGroup
	for _, link := range links {
		albumID := getAlbumID(link)

		wg.Add(1)
		go func(link string) {
			defer wg.Done()

			downloadSem <- struct{}{}
			defer func() { <-downloadSem }()

			parse(link, albumID)
		}(link)
	}

	wg.Wait()
}

func getAlbumID(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func readLinksFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var links []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		link := strings.TrimSpace(scanner.Text())
		if link != "" {
			links = append(links, link)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func parse(url string, albumID string) {
	downloadDir := "D:\\photos\\tmp\\" + albumID

	err := os.MkdirAll(downloadDir, 0755)
	if err != nil {
		fmt.Println("error: can not create dir: ", err)
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("error: can not get page: ", err)
		return
	}
	defer resp.Body.Close()

	imgLinks := extractImageLinks(resp.Body)

	for _, link := range imgLinks {
		downloadImage(downloadDir, link)
		time.Sleep(1 * time.Second) // sleeper maybe errors without
	}

	fmt.Printf("info: downloading imgs done: %s.\n", albumID)
}

func extractImageLinks(body io.Reader) []string {

	re := regexp.MustCompile(`<img[^>]+src="([^"]+)"`) //find img links

	var links []string
	buf := make([]byte, 1024)
	for {
		n, err := body.Read(buf)
		if err != nil {
			break
		}
		matches := re.FindAllStringSubmatch(string(buf[:n]), -1)
		for _, match := range matches {
			link := match[1]
			if isValidImageLink(link) {
				links = append(links, link)
			}
		}
	}
	return links
}

func isValidImageLink(link string) bool {
	return strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")
}

func downloadImage(dir, url string) {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("error: http.get: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("error: http status code not ok: ", resp.StatusCode)
		return
	}

	filename := path.Join(dir, getFilenameFromURL(url))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("error: can not create file: ", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Println("error: can not copy in file: ", err)
		return
	}

	fmt.Println("info: img saved: ", filename)
}

func getFilenameFromURL(url string) string {
	_, filename := path.Split(url)
	return filename
}
