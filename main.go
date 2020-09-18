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

	"github.com/asdine/storm"
	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
)

// Memory searched word
type Memory struct {
	ID        int    `storm:"id,increment"` // primary key
	English   string `storm:"unique"`       // this field will be indexed with a unique constraint
	Means     string
	CreatedAt time.Time `storm:"index"` // this field will be indexed
	Memorized bool
}

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

	DB *storm.DB
)

func init() {
	f, err := os.OpenFile("kodic.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func validateInput(input string) *string {
	word := strings.Trim(input, " ")
	if word == previousWord {
		return nil
	}
	previousWord = word

	if !wordRegExp.MatchString(word) {
		log.Println("wrong word: " + word)
		return nil
	}
	word = strings.ToLower(word)
	return &word
}

func askWord(word *string) []byte {
	searchURL := fmt.Sprintf(rootURL+"search?m=\"pc\"&query=\"%s\"", *word)
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

func saveWord2DB(word, means *string) {
	memory := Memory{
		English:   *word,
		Means:     *means,
		CreatedAt: time.Now(),
		Memorized: false,
	}

	err := DB.Save(&memory)
	if err != nil {
		log.Println("failed to save word: " + err.Error())
		return
	}
	log.Println("saved: " + *word)
}

func getMeansFromDB(word *string) *string {
	var memory Memory

	err := DB.One("English", *word, &memory)
	if err == storm.ErrNotFound {
		return nil
	} else if err != nil {
		log.Println("failed to search word: " + *word)
		return nil
	}

	return &memory.Means
}

func makeMeans(dictResponse *DictResponse) *string {
	means := dictResponse.SearchResultMap.SearchResultListMap.Word.Items[0].MeansCollectors[0].Means

	text := ""
	index := 1
	for _, mean := range means {
		newText := mean.Value
		newText = arrowRegExp.ReplaceAllString(newText, "")
		newText = equalRegExp.ReplaceAllString(newText, "")
		newText = biArrowRegExp.ReplaceAllString(newText, "")
		newText = tagRegExp.ReplaceAllString(newText, "")
		newText = strongRegExp.ReplaceAllString(newText, "")
		newText = abbrRegExp.ReplaceAllString(newText, "")
		if len(newText) != 0 {
			text += strconv.Itoa(index) + ". " + newText + " "
			index++
		}
	}

	return &text
}

func sendNotification(word, means *string) {
	beeep.Notify(*word, *means, "./icon.jpg")
}

func run() {
	defer time.Sleep(interval)

	input, err := clipboard.ReadAll()
	if err != nil {
		log.Println("cannot read clipboard : " + err.Error())
		return
	}

	word := validateInput(input)
	if word == nil {
		return
	}

	means := getMeansFromDB(word)
	if means == nil {
		body := askWord(word)
		if body == nil {
			return
		}

		dictResponse := parseResponse(body)
		if dictResponse == nil {
			return
		}

		means = makeMeans(dictResponse)

		saveWord2DB(word, means)
	}

	sendNotification(word, means)
}

func main() {
	db, err := storm.Open("kodic.db")
	if err != nil {
		log.Fatal(err)
	}
	DB = db
	defer DB.Close()

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
