package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
)

type Mean struct {
	Value string `json:"value,omitempty"`
}

type MeansCollector struct {
	Means []Mean `json:"means,omitempty"`
}

type Item struct {
	MeansCollectors []MeansCollector `json:"meansCollector,omitempty"`
}

type Word struct {
	Items []Item `json:"items,omitempty"`
}

type SearchResultListMap struct {
	Word Word `json:"WORD,omitempty"`
}

type SearchResultMap struct {
	SearchResultListMap SearchResultListMap `json:"searchResultListMap,omitempty"`
}

type NaverDictResponse struct {
	SearchResultMap SearchResultMap `json:"searchResultMap,omitempty"`
}

func main() {
	oldWord := ""
	for {
		defer time.Sleep(500 * time.Microsecond)

		newWord, err := clipboard.ReadAll()
		if err != nil {
			log.Fatal(err)
		}

		if newWord == oldWord {
			continue
		}
		oldWord = newWord

		url := fmt.Sprintf("https://en.dict.naver.com/api3/enko/search?m=\"pc\"&query=\"%s\"", newWord)
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}

		var ndr NaverDictResponse
		jsonErr := json.Unmarshal(body, &ndr)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}
		means := ndr.SearchResultMap.SearchResultListMap.Word.Items[0].MeansCollectors[0].Means

		var text string
		for _, mean := range means {
			text += mean.Value + " "
		}

		go beeep.Notify("Word: "+newWord, "Mean: "+text, "")
	}
}
