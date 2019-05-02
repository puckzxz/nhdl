package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gocolly/colly"
)

// TODO: Add progress bars?
// TODO: Add proxies to circumvent possible bans?

// Gallery holds data about the nhentai gallery and where to download it
type Gallery struct {
	ID         string 	// The ID of the gallery
	URL        string 	// The URL to the gallery
	Size       int 		// The amount of pages in the gallery
	Images     []string // The direct image URL's to all the images in the gallery
	FolderPath string 	// The folder on the drive where the gallery should be downloaded to
}

// GetSize gets the amount of images in the gallery
func (g *Gallery) GetSize() {
	fmt.Printf("Getting the size of the %s...\n", g.ID)
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})
	var pageURLS []string
	c.OnHTML(".gallerythumb", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		url := fmt.Sprintf("https://nhentai.net%s", link)
		pageURLS = append(pageURLS, url)
	})
	c.Visit(g.URL)
	c.Wait()
	g.Size = len(pageURLS)
}

// GetImages gets the direct image URL to all the images in the gallery
/*
	Unfortunately the slowest function in this program and needs improvement
	We have to visit every image in the gallery to get the direct link to the image
	As far as I know there are no ways to avoid this
	We can't just get the first image and use the server ID it's hosted on for all other images
	Ex. https://i.nhentai.net/galleries/<id>/<file>.jpg
	For larger galleries the ID will sometimes change
	The image extension will also sometimes change at random in the Gallery, we could have 1.jpg then 12.png
	With nhentai placing harsher restrictions on scraping we have to slow down even more so we don't get banned
*/
func (g *Gallery) GetImages() {
	fmt.Printf("Getting all the images in the Gallery...\n")
	fmt.Printf("This may take a bit...\n")
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", RandomDelay: 5 * time.Second})
	c.OnHTML(".fit-horizontal", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		g.Images = append(g.Images, link)
	})
	for i := 1; i <= g.Size; i++ {
		url := fmt.Sprintf("%s%d", g.URL, i)
		c.Visit(url)
		c.Wait()
	}
}

// Download downloads the gallery
func (g *Gallery) Download() {
	g.GetSize()
	g.GetImages()
	fmt.Printf("Downloading %d images from Gallery %s...\n", g.Size, g.ID)
	if _, err := os.Stat(g.FolderPath); os.IsNotExist(err) {
		err = os.Mkdir(g.FolderPath, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	wg.Add(len(g.Images))
	for _, image := range g.Images {
		fname := fmt.Sprintf(`%s\%s`, g.FolderPath, path.Base(image))
		go func(url string, filename string, wg *sync.WaitGroup) error {
			// Create the file on the disk
			file, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer file.Close()
			// Get the data from our URL
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			// Write the data to our file
			_, err = io.Copy(file, resp.Body)
			if err != nil {
				return err
			}
			defer wg.Done()
			return nil
		}(image, fname, &wg)
	}
	wg.Wait()
	fmt.Printf("Downloaded Gallery %s to %s\n", g.ID, g.FolderPath)
}

func main() {
	currentDir, _ := os.Getwd()
	id := flag.String("id", "", "The magic numbers of the Gallery you want to download. (Required)")
	dlPath := flag.String("path", currentDir, "Where to put the folder containing the files, defaults to the current directory.")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
	if *id == "" {
		log.Fatal("[!] You must supply the magic numbers!")
	}
	g := &Gallery{}
	g.ID = *id
	g.URL = fmt.Sprintf("https://nhentai.net/g/%s/", g.ID)
	g.FolderPath = fmt.Sprintf(`%s\%s`, *dlPath, g.ID)
	g.Download()
}
