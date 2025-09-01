package logic

import (
	"encoding/json"
	"fmt"

	"github.com/gpt-utils/internal/dto"
)

const anilistURL = "https://graphql.anilist.co"

// ...existing code...
type ResponseAnilist struct {
	Data struct {
		Media struct {
			ID    int `json:"id"`
			Title struct {
				Romaji  string `json:"romaji"`
				English string `json:"english"`
			} `json:"title"`
			Description     string `json:"description"`
			AverageScore    int    `json:"averageScore"`
			CountryOfOrigin string `json:"countryOfOrigin"`
			Episodes        int    `json:"episodes"`
			Format          string `json:"type"`
			StartDate       struct {
				Day   int `json:"day"`
				Month int `json:"month"`
				Year  int `json:"year"`
			} `json:"startDate"`
			EndDate struct {
				Day   int `json:"day"`
				Month int `json:"month"`
				Year  int `json:"year"`
			} `json:"endDate"`
			Status     string
			IsAdult    bool     `json:"isAdult"`
			Synonyms   []string `json:"synonyms"`
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
						DateOfBirth struct {
							Day   int `json:"day"`
							Month int `json:"month"`
							Year  int `json:"year"`
						} `json:"dateOfBirth"`
						Age  string `json:"age"`
						ID   int    `json:"id"`
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
		DateOfBirth dto.DateOfBirth
		Age         string `json:"age"`
		ID          int    `json:"id"`
		Name        struct {
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

// ...existing code...
type CombinedResult struct {
	FullResponse *ResponseAnilist `json:"fullResponse"`
	AllEdges     []CharacterEdge  `json:"allEdges"`
}

func fetchAnimeCharacters(search string, page, perPage int) (*ResponseAnilist, error) {
	query := `
    query ($search: String!, $page: Int = 1, $perPage: Int = 50) {
      Media(search: $search, type: ANIME, isAdult: false) {
        id
        title {
          romaji
          english
        }
        description
        averageScore
        countryOfOrigin
        episodes
        format
        startDate {
          day
          month
          year
        }
        endDate {
          day
          month
          year
        }
        isAdult
        synonyms
        characters(page: $page, perPage: $perPage) {
          pageInfo {
            currentPage
            lastPage
            perPage
            hasNextPage
          }
          edges {
             voiceActors {
				id
				name {
					full
				}
				image {
					large
				}
				language
				siteUrl
				}
			}
            node {
              dateOfBirth {
                day
                month
                year
              }
              age
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
	seen := make(map[int]bool)

	for {
		resp, err := fetchAnimeCharacters(search, page, perPage)
		if err != nil {
			return nil, nil, err
		}

		if fullResponse == nil {
			fullResponse = resp
		}

		// Converte para tipo simplificado
		for _, edge := range resp.Data.Media.Characters.Edges {

			if seen[edge.Node.ID] {
				continue // já temos esse personagem, pula
			}
			seen[edge.Node.ID] = true

			allEdges = append(allEdges, CharacterEdge{
				Role: edge.Role,
				Node: struct {
					DateOfBirth dto.DateOfBirth
					Age         string `json:"age"`
					ID          int    `json:"id"`
					Name        struct {
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
					DateOfBirth: dto.DateOfBirth{
						Day:   edge.Node.DateOfBirth.Day,
						Month: edge.Node.DateOfBirth.Month,
						Year:  edge.Node.DateOfBirth.Year,
					},
					Age: edge.Node.Age,
					ID:  edge.Node.ID,
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
