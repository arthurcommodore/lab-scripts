package logic

import (
	"encoding/json"
	"fmt"

	"github.com/gpt-utils/internal/logic/utils"
)

const anilistURL = "https://graphql.anilist.co"

type ResponseAnilist struct {
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

type CombinedResult struct {
	FullResponse *ResponseAnilist `json:"fullResponse"`
	AllEdges     []CharacterEdge  `json:"allEdges"`
}

func fetchAnimeCharacters(search string, page, perPage int) (*ResponseAnilist, error) {
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

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	req, err := HTTPPostWithHeaders(anilistURL, body, headers)
	if err != nil {
		return nil, err
	}

	utils.PrintResponse(req)

	var response ResponseAnilist
	err = json.Unmarshal(req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func FetchAllAnimeCharacters(search string, perPage int) ([]CharacterEdge, *ResponseAnilist, error) {
	page := 1
	var allEdges []CharacterEdge
	var fullResponse *ResponseAnilist

	for {
		resp, err := fetchAnimeCharacters(search, page, perPage)
		if err != nil {
			return nil, nil, err
		}

		if fullResponse == nil {
			fullResponse = resp
		} else {
			// Acumula os edges no fullResponse também
			fullResponse.Data.Media.Characters.Edges = append(
				fullResponse.Data.Media.Characters.Edges,
				resp.Data.Media.Characters.Edges...,
			)
		}

		// Converte para tipo simplificado
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

	return allEdges, fullResponse, nil
}
