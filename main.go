package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
)

// Mean holds word mean
type Mean struct {
	Value string `json:"value,omitempty"`
}

// MeansCollector holds means
type MeansCollector struct {
	Means []Mean `json:"means,omitempty"`
}

// Item holds meansCollectors
type Item struct {
	MeansCollectors []MeansCollector `json:"meansCollector,omitempty"`
}

// Word holds items
type Word struct {
	Items []Item `json:"items,omitempty"`
}

// SearchResultListMap holds word
type SearchResultListMap struct {
	Word Word `json:"WORD,omitempty"`
}

// SearchResultMap holds search result list map
type SearchResultMap struct {
	SearchResultListMap SearchResultListMap `json:"searchResultListMap,omitempty"`
}

// DictResponse holds search result map
type DictResponse struct {
	SearchResultMap SearchResultMap `json:"searchResultMap,omitempty"`
}

const (
	interval = time.Second
	rootURL  = "https://en.dict.naver.com/api3/enko/"
)

var (
	previousWord string
	regCode      *regexp.Regexp
)

func validateWord(word string) string {
	word = strings.Trim(word, " ")
	if word == previousWord {
		return ""
	}
	previousWord = word

	if !regCode.MatchString(word) {
		log.Println("wrong word: " + word)
		return ""
	}

	return word
}

func searchWord(word string) []byte {
	searchURL := fmt.Sprintf(rootURL+"search?m=\"pc\"&query=\"%s\"", word)
	resp, err := http.Get(searchURL)
	if err != nil {
		log.Println("HTTP request error: " + err.Error())
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read response: " + err.Error())
		return nil
	}

	return body
}

func parseResponse(body []byte) *DictResponse {
	var dictResponse DictResponse
	err := json.Unmarshal(body, &dictResponse)
	if err != nil {
		log.Println("failed to unmarshal: " + err.Error())
		return nil
	}
	if len(dictResponse.SearchResultMap.SearchResultListMap.Word.Items) == 0 {
		log.Println("empty result")
		return nil
	}

	return &dictResponse
}

func sendNotification(word string, dictResponse *DictResponse) {
	means := dictResponse.SearchResultMap.SearchResultListMap.Word.Items[0].MeansCollectors[0].Means

	var text string
	for _, mean := range means {
		text += mean.Value + " "
	}

	beeep.Notify("Word: "+word, "Mean: "+text, "./icon.jpg")
}

func run() {
	defer time.Sleep(interval)

	word, err := clipboard.ReadAll()
	if err != nil {
		log.Println("cannot read clipboard : " + err.Error())
		return
	}

	word = validateWord(word)
	if word == "" {
		return
	}

	body := searchWord(word)
	if body == nil {
		return
	}

	dictResponse := parseResponse(body)
	if dictResponse == nil {
		return
	}

	sendNotification(word, dictResponse)
}

func main() {
	previousWord = ""
	regCode, _ = regexp.Compile("^[a-zA-Z]+$")
	for {
		run()
	}
}
