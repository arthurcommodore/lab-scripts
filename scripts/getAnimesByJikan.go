package scripts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gpt-utils/internal/logic"
	"github.com/gpt-utils/internal/logic/utils"
)

type responseJikan struct {
	Pagination struct {
		Last_visible_page int `json:"last_visible_page"`
	} `json:"pagination"`
	Data []struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	} `json:"data"`
}

func GetAnimesJikan() {

	var page = 1
	for {
		resp, err := logic.HTTPGet(fmt.Sprintf("https://api.jikan.moe/v4/anime?page=%v", page))
		if err != nil {
			fmt.Printf("error: %v", err)
			return
		}

		var result responseJikan
		err = json.Unmarshal(resp, &result)
		if err != nil {
			fmt.Printf("%v", err)
		}

		if page > result.Pagination.Last_visible_page {
			return
		}

		data, err := json.Marshal(result.Data)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}

		fmt.Println(page)
		_, err = utils.SaveJSONToFileAppend(data, "jikan", "output")
		if err != nil {
			fmt.Printf("%v", err)
			return
		}

		time.Sleep(time.Second * 8)
		page++
	}
}
