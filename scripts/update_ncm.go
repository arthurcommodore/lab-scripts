package scripts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gpt-utils/internal/logic/gpt"
	"github.com/gpt-utils/internal/logic/utils"
)

type pieceJson struct {
	Code     string `json:"code"`
	PartName string `json:"partName"`
}

type pieceNcm struct {
	Code  string `json:"code"`
	Piece string `json:"piece"` // <-- isso deve bater com o JSON
	Ncm   string `json:"ncm"`
}

func UpdateNcm() {
	pieces, err := utils.LoadJSONFromFileAs[[]pieceJson]("/home/daym/servidorcaiu.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	var results []pieceNcm

	for _, piece := range pieces {
		prompt := fmt.Sprintf(
			"I need you to identify the correct NCM (Nomenclature of Mercosur Goods) for this part: %v. "+
				"Return your answer strictly in JSON format with the following fields: "+
				"`code` (the part code), `piece` (the exact part name as provided, do not change it), "+
				"and `ncm` (the NCM code). Do not include any text outside this JSON.",
			piece.PartName,
		)
		messages := []gpt.Message{
			{
				Role: "system",
				Content: `You are an expert in motorcycle parts and automotive NCM.
					Provide your answer strictly in JSON format, exactly matching this schema:
					{
					"code": "string",
					"piece": "string",
					"ncm": "string"
					}
					Do not include any text outside this JSON.`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		}

		jsonFormat := gpt.ResponseFormat{
			Type: "json_schema",
			Json_schema: gpt.JSONSchema{
				Name: "NCMResponse",
				Schema: gpt.Schema{
					Type: "object",
					Properties: map[string]interface{}{
						"code":  map[string]string{"type": "string"},
						"piece": map[string]string{"type": "string"},
						"ncm":   map[string]string{"type": "string"},
					},
					Required:             []string{"code", "piece", "ncm"},
					AdditionalProperties: false,
				},
			},
		}

		response, err := gpt.CallOpenAIStructOutPut(context.Background(), apiKey, "gpt-4.1-2025-04-14", messages, jsonFormat)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if len(response.Choices) == 0 || len(response.Choices[0].Message.Content) == 0 {
			fmt.Printf("Resposta GPT vazia para peça %s\n", piece.PartName)
			continue
		}

		var content pieceNcm
		if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &content); err != nil {
			fmt.Printf("Erro ao fazer parse do JSON para %s: %v\n", piece.PartName, err)
			continue
		}

		fmt.Println(content.Piece)
		results = append(results, content)
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Println("Erro ao converter resultados para JSON:", err)
		return
	}

	outputPath, err := utils.SaveJSONToFile(jsonData, "pieces_ncm", "/home/daym/output")
	if err != nil {
		fmt.Println("Erro ao salvar JSON:", err)
		return
	}

	fmt.Println("JSON salvo em:", outputPath)
}
