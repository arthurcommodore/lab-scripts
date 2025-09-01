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

type FtpClient struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

// cria cliente FTP e faz login
func NewFtpClient(addr, username, password string) (*FtpClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar: %w", err)
	}

	client := &FtpClient{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}

	// ler mensagem inicial
	client.readResp()

	client.sendCmd("USER " + username)
	client.readResp()

	client.sendCmd("PASS " + password)
	client.readResp()

	return client, nil
}

// envia comando e flush
func (c *FtpClient) sendCmd(cmd string) error {
	_, err := c.writer.WriteString(cmd + "\r\n")
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

// lê resposta
func (c *FtpClient) readResp() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// envia um arquivo dentro da sessão aberta
func (c *FtpClient) UploadFile(filepath string) error {
	// pedir PASV
	c.sendCmd("PASV")
	resp, err := c.readResp()
	if err != nil {
		return err
	}

	ip, port, err := parsePasvResponse(resp)
	if err != nil {
		return err
	}

	dataAddr := fmt.Sprintf("%s:%d", ip, port)
	dataConn, err := net.Dial("tcp", dataAddr)
	if err != nil {
		return fmt.Errorf("erro ao conectar dados: %w", err)
	}
	defer dataConn.Close()

	// enviar STOR
	filename := filepath[strings.LastIndex(filepath, "/")+1:]
	c.sendCmd("STOR " + filename)
	c.readResp()

	// abrir arquivo local
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// copiar para a conexão de dados
	_, err = io.Copy(dataConn, file)
	if err != nil {
		return err
	}

	// fechar dataConn e ler resposta final
	dataConn.Close()
	c.readResp()

	return nil
}

// encerra sessão
func (c *FtpClient) Close() error {
	c.sendCmd("QUIT")
	c.readResp()
	return c.conn.Close()
}

// parse da resposta PASV (mesmo que já tinha)
func parsePasvResponse(resp string) (string, int, error) {
	start := strings.Index(resp, "(")
	end := strings.Index(resp, ")")
	if start == -1 || end == -1 || end <= start {
		return "", 0, fmt.Errorf("resposta PASV mal formatada")
	}

	nums := strings.Split(resp[start+1:end], ",")
	if len(nums) != 6 {
		return "", 0, fmt.Errorf("resposta PASV inválida")
	}

	ip := strings.Join(nums[0:4], ".")
	p1, _ := strconv.Atoi(nums[4])
	p2, _ := strconv.Atoi(nums[5])
	port := p1*256 + p2

	return ip, port, nil
}
