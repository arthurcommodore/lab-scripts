package logic

import (
	"fmt"
)

type OpenAIRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

func CallOpenAI(apiKey, model, input string) ([]byte, error) {
	url := "https://api.openai.com/v1/responses"

	payload := OpenAIRequest{
		Model: model,
		Input: input,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
	}

	body, err := HTTPPostWithHeaders(url, payload, headers)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar OpenAI: %w", err)
	}

	return body, nil
}
