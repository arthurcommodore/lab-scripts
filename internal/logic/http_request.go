package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func HTTPPostWithHeaders(url string, payload interface{}, headers map[string]string) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao codificar JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar requisição: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// HTTPGet faz uma requisição GET e retorna o corpo da resposta
func HTTPGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição GET: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// HTTPPost faz uma requisição POST com JSON e retorna o corpo da resposta
func HTTPPost(url string, payload interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao codificar JSON: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erro na requisição POST: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
