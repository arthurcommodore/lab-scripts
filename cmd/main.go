package main

import (
	"log"
	"os"

	"github.com/gpt-utils/scripts"
)

func main() {

	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo de log: %v", err)
	}
	defer logFile.Close()

	// Redireciona o log padrão para o arquivo
	log.SetOutput(logFile)

	scripts.UpdateJustTypeAnimes()
}
