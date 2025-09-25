package gpt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gpt-utils/internal/logic"
)

func CallOpenAIStructOutPut(
	ctx context.Context,
	apiKey, model string,
	messages []Message,
	responseFormat ResponseFormat,
) (*GPTResp, error) {
	url := "https://api.openai.com/v1/chat/completions"

	payload := OpenAIRequest{
		Model:          model,
		Messages:       messages,
		ResponseFormat: responseFormat,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}

	body, err := logic.HTTPPostWithHeaders(url, payload, headers)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar OpenAI: %w", err)
	}
	var apiResp GPTResp

	if err := json.Unmarshal([]byte(body), &apiResp); err != nil {
		log.Fatalf("error parser json in gpt.go %v", err)
	}

	return &apiResp, nil
}
