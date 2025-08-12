package logic

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// parsePasvResponse extrai IP e porta da resposta PASV do servidor FTP
func parsePasvResponse(resp string) (string, int, error) {
	start := strings.Index(resp, "(")
	end := strings.Index(resp, ")")
	if start == -1 || end == -1 || end <= start {
		return "", 0, fmt.Errorf("resposta PASV mal formatada")
	}

	nums := strings.Split(resp[start+1:end], ",")
	if len(nums) != 6 {
		return "", 0, fmt.Errorf("resposta PASV com número incorreto de valores")
	}

	ip := strings.Join(nums[0:4], ".")
	p1, err := strconv.Atoi(nums[4])
	if err != nil {
		return "", 0, err
	}
	p2, err := strconv.Atoi(nums[5])
	if err != nil {
		return "", 0, err
	}

	port := p1*256 + p2
	return ip, port, nil
}

// SendFileFtpPasv envia arquivo via FTP no modo passivo (PASV) sem usar libs externas
func SendFileFtpPasv(addr, username, password, filepath string) error {
	// Conectar na porta de controle do FTP (21)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %w", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Função auxiliar para ler resposta do servidor
	readResp := func() (string, error) {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}

	// Ler mensagem inicial do servidor
	resp, err := readResp()
	if err != nil {
		return err
	}
	fmt.Println("Servidor:", resp)

	// USER
	_, err = writer.WriteString("USER " + username + "\r\n")
	if err != nil {
		return err
	}
	writer.Flush()
	resp, err = readResp()
	if err != nil {
		return err
	}
	fmt.Println("Servidor:", resp)

	// PASS
	_, err = writer.WriteString("PASS " + password + "\r\n")
	if err != nil {
		return err
	}
	writer.Flush()
	resp, err = readResp()
	if err != nil {
		return err
	}
	fmt.Println("Servidor:", resp)

	// PASV
	_, err = writer.WriteString("PASV\r\n")
	if err != nil {
		return err
	}
	writer.Flush()
	resp, err = readResp()
	if err != nil {
		return err
	}
	fmt.Println("Servidor:", resp)

	ip, port, err := parsePasvResponse(resp)
	if err != nil {
		return fmt.Errorf("erro ao parsear resposta PASV: %w", err)
	}
	fmt.Printf("Conectar para dados em %s:%d\n", ip, port)

	// Formatar endereço para conexão dados
	var dataAddr string
	if strings.Contains(ip, ":") { // IPv6 (raro em PASV)
		dataAddr = fmt.Sprintf("[%s]:%d", ip, port)
	} else { // IPv4
		dataAddr = fmt.Sprintf("%s:%d", ip, port)
	}

	dataConn, err := net.Dial("tcp", dataAddr)
	if err != nil {
		return fmt.Errorf("erro ao conectar na porta de dados: %w", err)
	}

	// Enviar comando STOR com nome do arquivo
	filename := filepath[strings.LastIndex(filepath, "/")+1:]
	_, err = writer.WriteString("STOR " + filename + "\r\n")
	if err != nil {
		return err
	}
	writer.Flush()
	resp, err = readResp()
	if err != nil {
		return err
	}
	fmt.Println("Servidor:", resp)

	// Abrir arquivo para enviar conteúdo
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo: %w", err)
	}
	defer file.Close()

	// Enviar arquivo pela conexão de dados
	buf := make([]byte, 4096)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, errw := dataConn.Write(buf[:n])
			if errw != nil {
				return fmt.Errorf("erro ao enviar dados: %w", errw)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("erro na leitura do arquivo: %w", err)
		}
	}

	dataConn.Close()

	// Ler resposta final do controle (upload concluído)
	resp, err = readResp()
	if err != nil {
		return err
	}
	fmt.Println("Servidor:", resp)

	// Enviar QUIT para encerrar sessão FTP
	_, err = writer.WriteString("QUIT\r\n")
	if err != nil {
		return err
	}
	writer.Flush()
	resp, err = readResp()
	if err != nil {
		return err
	}
	fmt.Println("Servidor:", resp)

	return nil
}
