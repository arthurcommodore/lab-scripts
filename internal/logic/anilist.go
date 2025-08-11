package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gpt-utils/internal/logic/utils"
)

const anilistURL = "https://graphql.anilist.co"

type Response struct {
	Data struct {
		Media struct {
			ID    int `json:"id"`
			Title struct {
				Romaji  string `json:"romaji"`
				English string `json:"english"`
			} `json:"title"`
			Characters struct {
				PageInfo struct {
					CurrentPage int  `json:"currentPage"`
					LastPage    int  `json:"lastPage"`
					PerPage     int  `json:"perPage"`
					HasNextPage bool `json:"hasNextPage"`
				} `json:"pageInfo"`
				Edges []struct {
					Role string `json:"role"`
					Node struct {
						ID   int `json:"id"`
						Name struct {
							Full   string `json:"full"`
							Native string `json:"native"`
						} `json:"name"`
						Image struct {
							Large  string `json:"large"`
							Medium string `json:"medium"`
						} `json:"image"`
						Description string `json:"description"`
						SiteURL     string `json:"siteUrl"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"characters"`
		} `json:"Media"`
	} `json:"data"`
}

func fetchAnimeCharacters(search string, page, perPage int) (*Response, error) {
	query := `
    query ($search: String!, $page: Int = 1, $perPage: Int = 50) {
      Media(search: $search, type: ANIME) {
        id
        title {
          romaji
          english
        }
        characters(page: $page, perPage: $perPage) {
          pageInfo {
            currentPage
            lastPage
            perPage
            hasNextPage
          }
          edges {
            role
            node {
              id
              name {
                full
                native
              }
              image {
                large
                medium
              }
              description
              siteUrl
            }
          }
        }
      }
    }
    `
	variables := map[string]interface{}{
		"search":  search,
		"page":    page,
		"perPage": perPage,
	}

	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", anilistURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody := new(bytes.Buffer)
	_, err = respBody.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	utils.PrintResponse(respBody.Bytes())

	var response Response
	err = json.Unmarshal(respBody.Bytes(), &response)
	if err != nil {
		return nil, err
	}

	utils.SaveJSONToFile(respBody.Bytes(), "aniList", "results")

	return &response, nil
}

type CharacterEdge struct {
	Role string `json:"role"`
	Node struct {
		ID   int `json:"id"`
		Name struct {
			Full   string `json:"full"`
			Native string `json:"native"`
		} `json:"name"`
		Image struct {
			Large  string `json:"large"`
			Medium string `json:"medium"`
		} `json:"image"`
		Description string `json:"description"`
		SiteURL     string `json:"siteUrl"`
	} `json:"node"`
}

func FetchAllAnimeCharacters(search string, perPage int) ([]CharacterEdge, error) {
	page := 1
	var allEdges []CharacterEdge

	for {
		resp, err := fetchAnimeCharacters(search, page, perPage)
		if err != nil {
			return nil, err
		}

		// Convertendo para seu tipo explícito CharacterEdge
		for _, edge := range resp.Data.Media.Characters.Edges {
			allEdges = append(allEdges, CharacterEdge{
				Role: edge.Role,
				Node: struct {
					ID   int `json:"id"`
					Name struct {
						Full   string `json:"full"`
						Native string `json:"native"`
					} `json:"name"`
					Image struct {
						Large  string `json:"large"`
						Medium string `json:"medium"`
					} `json:"image"`
					Description string `json:"description"`
					SiteURL     string `json:"siteUrl"`
				}{
					ID: edge.Node.ID,
					Name: struct {
						Full   string `json:"full"`
						Native string `json:"native"`
					}{
						Full:   edge.Node.Name.Full,
						Native: edge.Node.Name.Native,
					},
					Image: struct {
						Large  string `json:"large"`
						Medium string `json:"medium"`
					}{
						Large:  edge.Node.Image.Large,
						Medium: edge.Node.Image.Medium,
					},
					Description: edge.Node.Description,
					SiteURL:     edge.Node.SiteURL,
				},
			})
		}

		pageInfo := resp.Data.Media.Characters.PageInfo
		fmt.Printf("Fetched page %d of %d\n", pageInfo.CurrentPage, pageInfo.LastPage)

		if !pageInfo.HasNextPage || page >= pageInfo.LastPage {
			break
		}
		page++
	}

	return allEdges, nil
}
