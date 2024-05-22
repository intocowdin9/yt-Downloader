package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "views/index.html")
}

func YTSearchHandler(w http.ResponseWriter, r *http.Request) {
	videoURL := r.FormValue("yt-link")

	if videoURL == "" {
		http.Error(w, "Missing yt-link parameter", http.StatusBadRequest)
		return
	}

	extrctVideoID := func(videoURL string) (string, error) {
		const prefix = "https://www.youtube.com/watch?v="
		if strings.HasPrefix(videoURL, prefix) {
			return strings.TrimPrefix(videoURL, prefix), nil
		}

		return "", fmt.Errorf("invalid yt-link")
	}

	videoID, err := extrctVideoID(videoURL)
	if err != nil {
		http.Error(w, "invalid yt-link URL", http.StatusBadRequest)
		return
	}

	thumbnailURL := fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", videoID)
	cardHTML := fmt.Sprintf(`
	<div class="max-w-xs rounded overflow-hidden shadow-md bg-white p-2 my-8">
            <img id="thumbnail" class="w-full h-40 object-cover" src="%s" alt="Video Thumbnail">
            <div class="px-2 py-2">{{ Title }}
            </div>
            <div class="px-2 pb-1">
				<form action="yt-download" method="GET">
				<input type="hidden" name="link" value="%s">
					<button class="inline-block bg-black rounded-lg px-3 py-2 
					text-sm font-medium text-white mr-2 mb-2">Download</button>
				</form>
			</div>
        </div>`, thumbnailURL, videoURL)

	w.Write([]byte(cardHTML))
}

func YTDownloadHandler(w http.ResponseWriter, r *http.Request) {
	yt_link := r.FormValue("link")
	w.Write([]byte(yt_link))
	cmd := exec.Command("yt-dlp", yt_link)
	out, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Fatal(out)
	// w.Write(out)
}

func AboutUsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<h1 class="text-3xl sm:text-3xl lg:text-3xl text-white font-semibold leading-tight mb-4">No Promble :)</h1>`)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", HomeHandler).Methods("GET")
	r.HandleFunc("/search", YTSearchHandler).Methods("GET")
	r.HandleFunc("/about-us", AboutUsHandler).Methods("POST")
	r.HandleFunc("/yt-download", YTDownloadHandler).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:3000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
