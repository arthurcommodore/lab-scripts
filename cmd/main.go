package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gpt-utils/internal/logic"
	"github.com/gpt-utils/internal/logic/utils"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {

	ctx := context.Background()
	defer ctx.Done()

	if err := godotenv.Load(); err != nil {
		log.Fatal("Erro ao carregar arquivo .env:", err)
	}

	logic.Connect("mongodb://localhost:27017/animeSearch")

	apiKey := os.Getenv("OPENAI_API_KEY")

	client := logic.GetDB() // deve retornar *mongo.Client
	collection := client.Database("animeSearch").Collection("animes")

	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY não definido no ambiente")
		ctx.Done()
	}

	rep := logic.NewQueryAnimeMongo(collection)

	animes, err := rep.ListPageAnime(ctx, 1, 1, bson.M{})
	if err != nil {
		log.Fatal("ListPageAnime error")
		ctx.Done()
	}

	anime := animes[0]

	var charactersStr string
	for _, character := range anime.Characters {
		charactersStr += fmt.Sprintf(
			"name: '%s', bio: '%s'\n",
			character.Name,
			character.Bio,
		)
	}

	model := "gpt-4.1"
	input := fmt.Sprintf(
		"Preciso que você complemente os dados do anime: \"%s\".\n\n"+
			"Vou fornecer uma lista no formato:\n"+
			"campo1: 'valorAtual', campo2: 'valorAtual', ...\n"+
			"Para cada campo, você deve pesquisar e gerar um novo valor.\n"+
			"Se não encontrar informação confiável, mantenha exatamente o valor atual (mesmo que seja nil ou undefined).\n\n"+
			"No final, me retorne apenas um JSON válido, com aspas duplas corretas, contendo todos os campos com seus respectivos novos valores, para que eu possa atualizar meu banco de dados.\n"+
			"Não adicione explicações, apenas o JSON final.\n\n"+
			"Aqui estão os campos principais do anime:\n"+
			"synopsis: '%s'\n"+
			"status: '%s'\n"+
			"episodes: %d\n\n"+
			"Agora, preciso que você também gere um array chamado 'characters'.\n"+
			"Para o array 'characters', faça o seguinte:\n"+
			"- Se eu fornecer personagens, melhore as informações que eu já enviei (nome, bio, etc), corrigindo ou adicionando dados confiáveis.\n"+
			"- Procure por personagens adicionais relevantes desse anime e os adicione ao array.\n"+
			"- Se eu não fornecer nenhum personagem, crie o array com todos os personagens confiáveis que você encontrar.\n"+
			"- Se não encontrar nenhum personagem confiável, retorne um array vazio 'characters': [].\n\n"+
			" Dados dos Personages que já tenho: %s \n"+
			"Para terminar, adicione dois campos no JSON final:\n"+
			"- chatGpt: true\n"+
			"- chatGptDontFound: true, se você NÃO achou todos os dados confiáveis;\n"+
			"- chatGptDontFound: false, se você conseguiu encontrar todos os dados confiáveis.\n",
		anime.Title,
		anime.Synopsis,
		anime.Status,
		anime.Episodes,
		charactersStr, // pode estar vazio, tudo bem
	)

	response, err := logic.CallOpenAI(apiKey, model, input)
	if err != nil {
		log.Fatal("Erro ao chamar OpenAI:", err)
	}
	// Parsear a resposta JSON completa da API
	var apiResp struct {
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}

	if err := json.Unmarshal([]byte(response), &apiResp); err != nil {
		log.Fatalf("Erro ao parsear resposta completa da API: %v", err)
	}

	if len(apiResp.Output) == 0 || len(apiResp.Output[0].Content) == 0 {
		log.Fatal("Resposta do GPT não contém output esperado")
	}

	gptJsonStr := apiResp.Output[0].Content[0].Text

	// Remover crases ou espaços extras no início/fim
	gptJsonStr = strings.TrimSpace(gptJsonStr)
	gptJsonStr = strings.Trim(gptJsonStr, "`") // remove crases no início/fim, se houver

	// Agora tenta parsear
	var updatedData map[string]interface{}
	if err := json.Unmarshal([]byte(gptJsonStr), &updatedData); err != nil {
		log.Fatalf("Erro ao parsear JSON do GPT: %v", err)
	}

	// Fazer update no Mongo
	filter := bson.M{"title": anime.Title}
	update := bson.M{"$set": updatedData}

	count, err := rep.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Fatalf("Erro ao atualizar documento: %v", err)
	}

	fmt.Printf("Documentos atualizados: %d\n", count)

	filenamePrefix := "CallOpenAI-" + model
	outputDir := "results"
	filePath, err := utils.SaveJSONToFile(response, filenamePrefix, outputDir)
	if err != nil {
		log.Fatal("Erro ao salvar arquivo:", err)
	}

	log.Printf("Resposta salva em: %s\n", filePath)
}
