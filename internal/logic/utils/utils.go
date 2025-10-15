package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func SaveJSONToFileAppend(data []byte, filenamePrefix, outputDir string) (string, error) {
	// Caminho do arquivo
	filePath := filepath.Join(outputDir, filenamePrefix+".json")

	// Garante que o diretório existe
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("erro ao criar diretório: %w", err)
	}

	// Decodifica o novo JSON recebido
	var newData []interface{}
	if err := json.Unmarshal(data, &newData); err != nil {
		return "", fmt.Errorf("erro ao decodificar novo JSON: %w", err)
	}

	var existingData []interface{}

	// Se o arquivo já existe, lê o conteúdo e junta
	if _, err := os.Stat(filePath); err == nil {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("erro ao ler arquivo existente: %w", err)
		}

		if len(content) > 0 {
			if err := json.Unmarshal(content, &existingData); err != nil {
				return "", fmt.Errorf("erro ao decodificar JSON existente: %w", err)
			}
		}
	}

	// Faz o append dos novos dados
	existingData = append(existingData, newData...)

	// Regrava o JSON formatado
	finalData, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("erro ao serializar JSON final: %w", err)
	}

	if err := os.WriteFile(filePath, finalData, 0644); err != nil {
		return "", fmt.Errorf("erro ao escrever arquivo: %w", err)
	}

	return filePath, nil
}

func SaveJSONToFile(data []byte, filenamePrefix, outputDir string) (string, error) {
	// Decodifica o JSON para interface{}
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	// Re-encode com indentação
	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("erro ao formatar JSON: %w", err)
	}

	// Gera nome do arquivo
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("%s-%s.json", timestamp, filenamePrefix)

	// Cria diretório se não existir
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("erro ao criar diretório: %w", err)
	}

	fullPath := filepath.Join(outputDir, filename)

	// Escreve arquivo formatado
	if err := os.WriteFile(fullPath, prettyJSON, 0644); err != nil {
		return "", fmt.Errorf("erro ao salvar arquivo: %w", err)
	}

	return fullPath, nil
}

// PrintResponse imprime no console um []byte JSON formatado, ou imprime string pura se falhar
func PrintResponse(body []byte) {
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(body, &prettyJSON); err == nil {
		// Se for JSON válido, imprime formatado
		indented, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Println(string(indented))
	} else {
		// Caso contrário, imprime como string crua
		fmt.Println(string(body))
	}
}

func PrintJson(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ") // MarshalIndent deixa o JSON bonito
	if err != nil {
		log.Printf("erro ao converter para JSON: %v", err)
		return ""
	}
	return string(b)
}
