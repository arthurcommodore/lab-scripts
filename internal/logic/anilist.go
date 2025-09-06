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
			ID      int `json:"id"`
			Title   Title
			Studios struct {
				Nodes []struct {
					Name    string
					SiteUrl string
				}
			}
			Description       string `json:"description"`
			AverageScore      int    `json:"averageScore"`
			CountryOfOrigin   string `json:"countryOfOrigin"`
			Source            string
			Duration          int
			Episodes          int    `json:"episodes"`
			Format            string `json:"type"`
			StreamingEpisodes []StreamingEpisode
			StartDate         struct {
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
					Role        string `json:"role"`
					VoiceActors []VoiceActor
					Node        struct {
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

type StreamingEpisode struct {
	Site      string
	Thumbnail string
	Title     string
	Url       string
}

type VoiceActor struct {
	name struct {
		full string
	}
	image struct {
		large string
	}
	languageV2  string
	siteUrl     string
	homeTown    string
	gender      string
	age         int
	dateOfBirth struct {
		day   int
		month int
		year  int
	}
	dateOfDeath struct {
		day   int
		month int
		year  int
	}
}
type CharacterEdge struct {
	Role        string `json:"role"`
	VoiceActors []VoiceActor
	Node        struct {
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
type CombinedResultAniList struct {
	FullResponse *ResponseAnilist `json:"fullResponse"`
	AllEdges     []CharacterEdge  `json:"allEdges"`
}

type Title struct {
	English       string
	Native        string
	Romaji        string
	UserPreferred string
}

func fetchAnimeCharacters(search string, page, perPage int) (*ResponseAnilist, error) {
	query := `
    query ($search: String!, $page: Int = 1, $perPage: Int = 50) {
      Media(search: $search, type: ANIME, isAdult: false) {
        id
        title {
			romaji
			english
			userPreferred
			native
		}
		studios {
			nodes {
				name
				siteUrl
			}
		}
        description
        averageScore
        countryOfOrigin
		source
		duration
        episodes
        format
		streamingEpisodes {
			site
			thumbnail
			title
			url 
		}
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
				name {
					full
				}
				image {
					large
				}
				languageV2
				siteUrl
				homeTown
				gender
				age
				dateOfBirth {
					day
					month
					year 
				}
				dateOfDeath {
					day
					month
					year
				}
        	}
            node {
              dateOfBirth {
                day
                month
                year
              }
              age
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
				continue
			}
			seen[edge.Node.ID] = true

			allEdges = append(allEdges, CharacterEdge{
				Role:        edge.Role,
				VoiceActors: edge.VoiceActors,
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
