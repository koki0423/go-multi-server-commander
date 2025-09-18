package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// Server 構造体: YAMLファイル内の各サーバー情報を格納
type Server struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Config 構造体: YAMLファイル全体の構造に対応
type Config struct {
	Servers []Server `yaml:"servers"`
}

func main() {
	// --- 1. コマンドライン引数から実行するコマンドを取得 ---
	if len(os.Args) < 2 {
		fmt.Println("使用法: go run run-command.go \"<実行したいコマンド>\"")
		fmt.Println("例: go run run-command.go \"hostname\"")
		fmt.Println("例: go run run-command.go \"ls -l /tmp\"")
		os.Exit(1)
	}
	commandToRun := os.Args[1]

	// --- 2. YAMLファイルの読み込み ---
	yamlFile, err := os.ReadFile("servers.yaml")
	if err != nil {
		log.Fatalf("YAMLファイルの読み込みに失敗しました: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		log.Fatalf("YAMLの解析に失敗しました: %v", err)
	}

	// --- 3. 各サーバーでコマンドを並列実行 ---
	var wg sync.WaitGroup
	for _, server := range config.Servers {
		wg.Add(1)
		// goキーワードで各サーバーへの処理を並行して実行
		go func(s Server) {
			defer wg.Done()
			executeCommand(s, commandToRun)
		}(server)
	}
	// すべてのサーバーの処理が終わるのを待つ
	wg.Wait()
}

// executeCommand: 1台のサーバーでコマンドを実行し、結果を出力する
func executeCommand(server Server, command string) {
	// --- SSHクライアントの設定 ---
	clientConfig := &ssh.ClientConfig{
		User:            server.User,
		Auth:            []ssh.AuthMethod{ssh.Password(server.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)

	// --- SSH接続 ---
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		log.Printf("--- [%s] 接続失敗 ---\nError: %v\n", server.Host, err)
		return
	}
	defer client.Close()

	// --- 新しいSSHセッションを作成 ---
	session, err := client.NewSession()
	if err != nil {
		log.Printf("--- [%s] セッション作成失敗 ---\nError: %v\n", server.Host, err)
		return
	}
	defer session.Close()

	// --- コマンドを実行し、標準出力と標準エラー出力をまとめて取得 ---
	output, err := session.CombinedOutput(command)
	
	// --- 結果を出力 ---
	fmt.Printf("--- [Result from %s] ---\n", server.Host)
	if err != nil {
		// コマンド実行に失敗した場合
		fmt.Printf("Status: Command Failed\n")
		fmt.Printf("Error: %v\n", err)
	} else {
		// コマンド実行に成功した場合
		fmt.Printf("Status: OK\n")
	}
	// 実行結果の出力（成功・失敗問わず）
	fmt.Println(strings.TrimSpace(string(output)))
	fmt.Println("---") // 区切り線
}