package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
	interval = 500 * time.Millisecond
	rootURL  = "https://en.dict.naver.com/api3/enko/"
)

var (
	previousWord  string
	wordRegExp    *regexp.Regexp
	tagRegExp     *regexp.Regexp
	strongRegExp  *regexp.Regexp
	arrowRegExp   *regexp.Regexp
	equalRegExp   *regexp.Regexp
	biArrowRegExp *regexp.Regexp
	abbrRegExp    *regexp.Regexp
)

func init() {
	f, err := os.OpenFile("kodic.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func validateWord(word string) string {
	word = strings.Trim(word, " ")
	if word == previousWord {
		return ""
	}
	previousWord = word

	if !wordRegExp.MatchString(word) {
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
	index := 1
	for _, mean := range means {
		newText := mean.Value
		newText = tagRegExp.ReplaceAllString(newText, "")
		newText = strongRegExp.ReplaceAllString(newText, "")
		newText = arrowRegExp.ReplaceAllString(newText, "")
		newText = equalRegExp.ReplaceAllString(newText, "")
		newText = biArrowRegExp.ReplaceAllString(newText, "")
		newText = abbrRegExp.ReplaceAllString(newText, "")
		if len(newText) != 0 {
			text += strconv.Itoa(index) + ". " + newText + " "
			index++
		}
	}

	beeep.Notify(word, text, "./icon.jpg")
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
	wordRegExp, _ = regexp.Compile(`^[a-zA-Z]+$`)
	tagRegExp, _ = regexp.Compile(`</?span[^>]*>`)
	strongRegExp, _ = regexp.Compile(`</?strong[^>]*>`)
	arrowRegExp, _ = regexp.Compile(`\(→(.*?)\)`)
	equalRegExp, _ = regexp.Compile(`\(=(.*?)\)`)
	biArrowRegExp, _ = regexp.Compile(`\(↔(.*?)\)`)
	abbrRegExp, _ = regexp.Compile(`\(Abbr.\)`)
	for {
		run()
	}
}
