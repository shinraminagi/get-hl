package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const adapose = false
const number_of_frontends = 2

var intervalFlag = flag.Float64("interval", 1, "Interval between each download (sec)")

var errLimitReached = errors.New("You have temporarily reached the limit for how many images you can browse. See http://ehgt.org/g/509.gif for more details")

var httpClient *http.Client

func init() {
	//	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	httpClient = &http.Client{
		//		Jar: jar,
	}
}

func main() {
	flag.Parse()
	url := flag.Arg(0)

	fmt.Printf("Scraping %s...", url)
	list, err := getImageList(url)
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Println("done")
	fmt.Printf("Found %d images.\n", len(list))

	for len(list) != 0 {
		imgUrl := list[0]
		fmt.Printf("Downloading %s...", imgUrl)
		err := download(imgUrl)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Retry...")
		} else {
			fmt.Println("done")
			list = list[1:]
		}
		if *intervalFlag > 0 {
			fmt.Printf("Waiting for %f seconds...", *intervalFlag)
			time.Sleep(time.Duration(*intervalFlag) * time.Second)
			fmt.Println("OK.")
		}
	}
}

func getImageList(url string) ([]string, error) {
	m := regexp.MustCompile(`^(?:(?:https?:)?//)?hitomi.la/reader/\d+\.html(?:#.*)?`).FindStringSubmatch(url)
	if m == nil {
		m = regexp.MustCompile(`^(?:(?:https?:)?//)?hitomi.la/galleries/(\d+)\.html`).FindStringSubmatch(url)
		if m == nil {
			return nil, fmt.Errorf("Invalid hitomi.la URL: %s", url)
		}
		url = fmt.Sprintf(`https://hitomi.la/reader/%s.html`, m[1])
	}

	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return nil, err
	}

	list := []string{}
	doc.Find(`div.img-url`).Each(func(_ int, el *goquery.Selection) {
		list = append(list, url_from_url(el.Text(), ""))
	})

	return list, nil
}

var reReplace = regexp.MustCompile(`//..?\.hitomi\.la/`)

func url_from_url(url, base string) string {
	return "https:" + reReplace.ReplaceAllLiteralString(url, fmt.Sprintf(`//%s.hitomi.la/`, subdomain_from_url(url, base)))
}

func subdomain_from_url(url, base string) string {
	retval := "a"
	if base != "" {
		retval = base
	}

	m := regexp.MustCompile(`/\d*(\d)/`).FindStringSubmatch(url)
	if m == nil {
		return retval
	}

	g, err := strconv.ParseInt(m[1], 10, 32)
	if err != nil {
		return retval
	}

	if g == 1 {
		g = 0
	}

	retval = subdomain_from_galleryid(rune(g)) + retval

	return retval
}

func subdomain_from_galleryid(g rune) string {
	if adapose {
		return "0"
	}

	o := g % number_of_frontends

	return string([]rune{97 + o})
}

func download(rawurl string) error {
	filename, err := fileNameOf(rawurl)
	if err != nil {
		return err
	}
	resp, err := http.Get(rawurl)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

var reInPath = regexp.MustCompile("[^/]+$")

func fileNameOf(rawurl string) (string, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	file := reInPath.FindString(url.Path)
	if file == "" {
		return "", fmt.Errorf("Filename not found: %s", rawurl)
	}
	return file, nil
}
