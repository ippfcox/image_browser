package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"image"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
)

//go:embed image_list.html
var imageBrowseHtml embed.FS

const (
	kPageSize   = 30
	kThumbWidth = 180
)

var dir, reverseProxyAddr string
var port int
var reverseProxy *httputil.ReverseProxy
var supportedExts = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp"}
var imageInfos []imageInfo

func main() {
	flag.StringVar(&dir, "dir", ".", "image dir")
	flag.IntVar(&port, "port", 8000, "listen port")
	flag.StringVar(&reverseProxyAddr, "rproxy", "http://127.0.0.1:7860", "reverse proxy addr")
	flag.Parse()

	// dir
	dirfi, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		log.Fatalf("<%s> IsNotExist", dir)
	}
	if !dirfi.IsDir() {
		log.Fatalf("<%s> !IsDir", dir)
	}
	absdir, _ := filepath.Abs(dir)
	log.Printf("Serve dir: <%s>", absdir)
	watchImages(absdir)
	// reverse proxy
	target, err := url.Parse(reverseProxyAddr)
	if err == nil {
		reverseProxy = httputil.NewSingleHostReverseProxy(target)
		log.Printf("Reverse proxy addr: <%s>", reverseProxyAddr)
	}

	http.HandleFunc("/", reverseProxyHandler)
	http.HandleFunc("/ib/", imageBrowseHandler)
	http.HandleFunc("/ib/thumb/", thumbHandler)
	http.HandleFunc("/ib/image/", imageHandler)
	log.Printf("Listen and serve on :%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func reverseProxyHandler(w http.ResponseWriter, r *http.Request) {
	if reverseProxy == nil {
		http.Error(w, "No valid reverse proxy", http.StatusInternalServerError)
		return
	}
	r.Header.Set("X-Real-IP", r.RemoteAddr)
	reverseProxy.ServeHTTP(w, r)
}

func imageBrowseHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.FormValue("page"))

	pageRange := 1
	pageCount := (len(imageInfos) + kPageSize - 1) / kPageSize
	// clip
	page = Clip(page, 1, pageCount)        // current page
	prevPage := Clip(page-1, 1, pageCount) // previous page
	nextPage := Clip(page+1, 1, pageCount) // next page
	showFirstPage := page-pageRange > 1
	showLastPage := page+pageRange < pageCount

	pagesInRange := make([]int, 0) // page range
	for i := page - pageRange; i <= page+pageRange; i++ {
		if i >= 1 && i <= pageCount {
			pagesInRange = append(pagesInRange, i)
		}
	}

	start := (page - 1) * kPageSize
	end := start + kPageSize
	if end > len(imageInfos) {
		end = len(imageInfos)
	}

	tmpl, err := template.ParseFS(imageBrowseHtml, "image_list.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Page":          page,         // current page
		"FirstPage":     1,            // first page
		"LastPage":      pageCount,    // last page
		"PrevPage":      prevPage,     // previous page
		"NextPage":      nextPage,     // nest page
		"PagesInRange":  pagesInRange, // page range
		"ShowFirstPage": showFirstPage,
		"ShowLastPage":  showLastPage,
		"Images":        imageInfos[start:end],
	}
	if err = tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func thumbHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	imgInfo := imageInfos[Clip(id, 0, len(imageInfos)-1)]
	f, err := os.Open(imgInfo.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	img, err := imaging.Decode(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	imgWidth := img.Bounds().Max.X
	imgHeight := img.Bounds().Max.Y
	var cropRect image.Rectangle
	if imgWidth > imgHeight {
		cropRect = image.Rect((imgWidth-imgHeight)/2, 0, imgHeight, imgHeight)
	} else {
		cropRect = image.Rect(0, 0, imgWidth, imgWidth)
	}
	thumb := imaging.Resize(imaging.Crop(img, cropRect), kThumbWidth, kThumbWidth, imaging.Lanczos)
	imaging.Encode(w, thumb, imaging.JPEG)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	imgInfo := imageInfos[Clip(id, 0, len(imageInfos)-1)]
	f, err := os.Open(imgInfo.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	io.Copy(w, f)
}
