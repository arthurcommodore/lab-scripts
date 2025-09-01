package scripts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gpt-utils/internal/logic"
	"github.com/otiai10/gosseract/v2"
)

// converte PDF em imagens PNG (uma por página)
func pdfToImages(pdfPath, outputPrefix string) ([]string, error) {
	cmd := exec.Command("pdftoppm", "-png", pdfPath, outputPrefix)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("erro ao converter PDF em imagens: %w", err)
	}

	// lista todos arquivos gerados (out-1.png, out-2.png, etc)
	files, err := filepath.Glob(outputPrefix + "-*.png")
	if err != nil {
		return nil, fmt.Errorf("erro ao listar imagens: %w", err)
	}
	return files, nil
}

// roda OCR no arquivo de imagem e retorna o texto
func ocrImage(imgPath string, client *gosseract.Client) (string, error) {
	err := client.SetImage(imgPath)
	if err != nil {
		return "", fmt.Errorf("erro ao carregar imagem %s: %w", imgPath, err)
	}
	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("erro no OCR da imagem %s: %w", imgPath, err)
	}
	return text, nil
}

type Vehicle struct {
	NF string `json:"nf"` // campo exportado e tag JSON
}

func downloadImages() {
	data, err := ioutil.ReadFile("/home/daym/sgl-db-latest.vehicles.json")
	if err != nil {
		log.Println(err)
	}

	var vehicles []Vehicle

	err = json.Unmarshal(data, &vehicles)
	if err != nil {
		log.Println(err)
	}

	// Exemplo: imprimir todos os NF
	for _, v := range vehicles {
		path := v.NF
		filename := filepath.Base(path)
		name := strings.TrimSuffix(filename, filepath.Ext(filename))
		if name == "" {
			continue
		}

		err := logic.DownloadImage(fmt.Sprintf("https://assets.locavibe.com.br/%s", v.NF), fmt.Sprintf("/home/daym/Documentos/nfVehicles/%s", name))
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(name)
		time.Sleep(2000)
	}
}

func RunScrapPdfVehicle() {

	files, err := filepath.Glob("/home/daym/Documentos/nfVehicles/*")
	if err != nil {
		log.Println(err)
	}

	type VehicleData struct {
		NF            string  `json:"nf"`
		PurchasePrice float64 `json:"purchasePrice"`
	}

	var data []VehicleData

	client := gosseract.NewClient()
	defer client.Close()
	client.SetLanguage("por")

	for _, f := range files {
		pdfName := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))

		images, err := pdfToImages(f, pdfName)
		if err != nil {
			log.Println("Erro ao converter PDF:", f, err)
			continue
		}
		if len(images) == 0 {
			log.Println("Nenhuma imagem gerada para PDF:", f)
			continue
		}

		var builder strings.Builder
		for _, img := range images {
			fmt.Println("Processando:", img)
			text, err := ocrImage(img, client)
			if err != nil {
				log.Println("Erro no OCR da imagem:", img, err)
				continue
			}
			builder.WriteString(text)
			builder.WriteString("\n---\n")
		}

		result := builder.String()
		purchasePrice, err := getMaxPrice(result)
		if err != nil {
			log.Println("Nenhum valor encontrado em:", f)
			continue
		}

		data = append(data, VehicleData{NF: pdfName, PurchasePrice: purchasePrice})
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Println("Erro ao serializar JSON:", err)
		return
	}

	err = ioutil.WriteFile("saida.json", jsonData, 0644)
	if err != nil {
		log.Println("Erro ao salvar saída:", err)
		return
	}

	fmt.Println("✅ Resultado salvo em saida.json")
}

func getMaxPrice(text string) (float64, error) {
	// Limpa quebras de linha e múltiplos espaços
	cleanText := strings.Join(strings.Fields(text), " ")

	// Regex para capturar todos os valores R$ do texto
	priceRegex := regexp.MustCompile(`R\$\s?([\d\.]+,\d{2})`)
	priceMatches := priceRegex.FindAllStringSubmatch(cleanText, -1)
	if len(priceMatches) == 0 {
		return 0, fmt.Errorf("nenhum valor encontrado")
	}

	var maxPrice float64
	for _, match := range priceMatches {
		priceStr := strings.ReplaceAll(match[1], ".", "")
		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		priceFloat, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			continue
		}
		if priceFloat > maxPrice {
			maxPrice = priceFloat
		}
	}

	return maxPrice, nil
}
