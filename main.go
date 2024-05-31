package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
)

type Client struct {
	ClientName        string `json:"clientName"`
	ClientVersion     string `json:"clientVersion"`
	AndroidSdkVersion int    `json:"androidSdkVersion"`
}

type Context struct {
	Client Client `json:"client"`
}

type RequestData struct {
	VideoId string  `json:"videoId"`
	Context Context `json:"context"`
}

type Response struct {
	StreamingData struct {
		Formats         []Format         `json:"formats"`
		AdaptiveFormats []AdaptiveFormat `json:"adaptiveFormats"`
	} `json:"streamingData"`
	VideoDetails struct {
		Title     string    `json:"title"`
		Thumbnail Thumbnail `json:"thumbnail"`
	} `json:"videoDetails"`
}

type Format struct {
	URL          string `json:"url"`
	QualityLabel string `json:"qualityLabel"`
	MimeType     string `json:"mimeType"`
}

type AdaptiveFormat struct {
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
	Bitrate  int    `json:"bitrate"`
}

type Thumbnail struct {
	Thumbnails []ThumbnailDetail `json:"thumbnails"`
}

type ThumbnailDetail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "views/index.html")
}

func YTSearchHandler(w http.ResponseWriter, r *http.Request) {
	videoURL := r.FormValue("yt-link")

	u, err := url.Parse(videoURL)
	if err != nil {
		http.Error(w, "error parsing yt-URL", http.StatusInternalServerError)
		return
	}

	queryParams := u.Query()
	videoID := queryParams.Get("v")
	if videoID == "" {
		http.Error(w, "no video ID found in yt-URL", http.StatusNotFound)
		return
	}

	data := RequestData{
		VideoId: videoID,
		Context: Context{
			Client: Client{
				ClientName:        "ANDROID_TESTSUITE",
				ClientVersion:     "1.9",
				AndroidSdkVersion: 30,
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "error marshaling JSON", http.StatusInternalServerError)
		return
	}

	apiURL := "https://www.youtube.com/youtubei/v1/player?key=AIzaSyA8eiZmM1FaDVjRy-df2KTyQ_vz_yYM39w"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "error creating request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "error sending request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "error reading response body", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("request failed with status code: %d\n%s", resp.StatusCode, body), resp.StatusCode)
		return
	}

	// Parse the JSON response
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Extract the thumbnail URL with height 180
	var thumbnailURL string

	for _, thumbnail := range response.VideoDetails.Thumbnail.Thumbnails {
		if thumbnail.Height == 180 && thumbnail.Width == 320 {
			thumbnailURL = thumbnail.URL
		}

		if thumbnailURL != "" {
			break
		}
	}

	maxThumbnailURL := fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoID)

	var bestAudioURL *string
	var bestBitrate int

	for _, format := range response.StreamingData.AdaptiveFormats {
		if format.MimeType == `audio/webm; codecs="opus"` || format.MimeType == `audio/mp4; codecs="mp4a.40.2"` {
			if bestAudioURL == nil || format.Bitrate > bestBitrate {
				bestAudioURL = &format.URL
				bestBitrate = format.Bitrate
			}
		}
	}

	// if bestAudioURL != nil {
	// 	fmt.Printf("Best Audio URL: %s\n", *bestAudioURL)
	// } else {
	// 	fmt.Println("Audio URL not found")
	// }

	// Print the URL
	if len(response.StreamingData.Formats) > 0 {

		cardHTML := fmt.Sprintf(
			`<div class="max-w-sm mx-auto">
			<div class="bg-white shadow-sm rounded-sm overflow-hidden">
				<img class="w-full h-38 object-cover" src="%s" alt="Card image">
				<div class="p-4">
				<h3 class="text-lg font-semibold">%s</h3><br>
				<div class="flex flex-col md:flex-row space-y-2 md:space-y-0 md:space-x-2">
					<a href="%s" target="_blank" download class="bg-gradient-to-r from-purple-400 via-pink-500 to-red-500 hover:bg-gradient-to-br text-white font-bold py-2 px-4 rounded-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-purple-300">Video %s</a>
					<a href="%s" target="_blank" download class="bg-gradient-to-r from-purple-400 via-blue-500 to-red-500 hover:bg-gradient-to-br text-white font-bold py-2 px-4 rounded-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-yellow-600">Mp3</a>
					<a href="%s" download class="bg-gradient-to-r from-purple-400 via-blue-500 to-red-500 hover:bg-gradient-to-br text-white font-bold py-2 px-4 rounded-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-yellow-600">Thumbnail</a>
				</div>
			</div>
			</div>
		</div>
			
			`,
			thumbnailURL,
			response.VideoDetails.Title,
			response.StreamingData.Formats[len(response.StreamingData.Formats)-1].URL,
			response.StreamingData.Formats[len(response.StreamingData.Formats)-1].QualityLabel,
			*bestAudioURL,
			maxThumbnailURL,
		)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cardHTML))
	} else {
		fmt.Println("No URL found in the response")
	}
}

func main() {
	r := mux.NewRouter()
	staticDir := "/static/"

	r.HandleFunc("/", HomeHandler).Methods("GET")
	r.HandleFunc("/search", YTSearchHandler).Methods("GET")
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("static"))))

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:5000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
