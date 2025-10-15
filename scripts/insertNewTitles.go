package scripts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/bson"
)

func InserNewTitles() {
	ctx := context.Background()
	filePath := "output/jikan.json"

	// Lê o conteúdo do arquivo
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Erro ao ler arquivo:", err)
		return
	}

	// Decodifica o JSON
	var jikanJson []struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(content, &jikanJson); err != nil {
		fmt.Println("Erro ao decodificar JSON:", err)
		return
	}

	// Monta um mapa para busca rápida de títulos existentes
	existingTitles := make(map[string]bool)

	// Loop pelas páginas do repositório
	page := 1
	for {
		animes, err := rep.ListPageAnime(ctx, page, 1000000, bson.M{})
		if err != nil {
			fmt.Println("Erro ao listar animes:", err)
			return
		}

		if len(animes) == 0 {
			break // não há mais resultados
		}

		// Adiciona títulos ao mapa
		for _, anime := range animes {
			existingTitles[anime.Title] = true
		}

		page++
	}

	// Cria uma lista com apenas os títulos que ainda não existem
	var newTitles []string
	for _, jk := range jikanJson {
		if !existingTitles[jk.Title] {
			newTitles = append(newTitles, jk.Title)
		}
	}

	// Exibe os resultados
	fmt.Printf("Total de novos títulos: %d\n", len(newTitles))
	for _, title := range newTitles {
		fmt.Println(title)
	}

	// Aqui você pode inserir os novos títulos no banco
	// ex: rep.InsertMany(ctx, newTitles)
}
