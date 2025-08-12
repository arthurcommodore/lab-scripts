package scripts

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gpt-utils/internal/logic"
	"github.com/gpt-utils/internal/logic/utils"
	"github.com/joho/godotenv"
)

func RequestAnimeList() {
	allEdges, fullResponse, err := logic.FetchAllAnimeCharacters("Naruto", 25)
	if err != nil {
		log.Fatalf("Erro: %v", err)
	}

	combined := logic.CombinedResult{
		FullResponse: fullResponse,
		AllEdges:     allEdges,
	}

	// Converte os dois juntos para JSON formatado
	jsonData, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		log.Fatalf("Erro ao converter para JSON: %v", err)
		return
	}

	filePath, err := utils.SaveJSONToFile(jsonData, "naruto-characters", "./output")
	if err != nil {
		log.Fatalf("Erro ao salvar arquivo: %v", err)
	}

	fmt.Println("Arquivo salvo em:", filePath)

	err = logic.DownloadImage(combined.AllEdges[0].Node.Image.Large, "output/golang.jpg")
	if err != nil {
		log.Fatal(err)
		return
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	addr := os.Getenv("FTP_ADDR")
	user := os.Getenv("FTP_USER")
	password := os.Getenv("FTP_PASSWORD")

	err = logic.SendFileFtpPasv(addr, user, password, "output/golang.jpg")
	if err != nil {
		fmt.Println("Erro:", err)
	} else {
		fmt.Println("Upload concluído com sucesso!")
	}

}
