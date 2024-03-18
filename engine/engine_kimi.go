package engine

import (
	"bytes"
	"chatgpt-bot/cfg"
	"chatgpt-bot/model"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type KimiEngine struct {
	client  *http.Client
	baseUrl string
	alive   bool

	apiKey string
}

func NewKimiEngine() *KimiEngine {
	return &KimiEngine{
		client: &http.Client{},
	}
}

func (e *KimiEngine) Init(cfg *cfg.Config) error {
	e.client = &http.Client{}
	e.baseUrl = fmt.Sprintf("http://%s:%d", cfg.EngineConfig.Host, cfg.EngineConfig.Port)
	e.apiKey = cfg.EngineConfig.ApiKey
	go e.checkKimiEngine()
	return nil
}

func (e *KimiEngine) Chat(ctx model.ChatContext) (string, error) {
	return e.chat2Kimi(ctx)
}

func (e *KimiEngine) Alive() bool {
	return true
}

func (e *KimiEngine) checkKimiEngine() {
	for {
		status := e.Alive()
		if !status {
			log.Println("[HealthCheck] chatgpt engine is not ready")
			e.alive = false
		} else {
			e.alive = true
		}
		time.Sleep(10 * time.Second)
	}
}

func (e *KimiEngine) chat2Kimi(ctx model.ChatContext) (string, error) {
	// 暂时不做session管理
	sentence, userID, _ := ctx.Question, ctx.UserID, ctx.Model
	log.Printf("[Kimi][%s] send request to kimi, text: %s\n", userID, sentence)
	choice, err := e.ChatCompletions(sentence)
	if err != nil {
		return "", err
	}
	log.Printf("[Kimi][%s] get response from kimi, text: %s \n", userID, choice)

	return choice, nil
}

// ChatRequest 定义了发送给OpenAI的请求体结构
type ChatRequest struct {
	Messages  []Message `json:"messages"`
	UseSearch bool      `json:"use_search"`
	Stream    bool      `json:"stream"`
}

// Message 定义了请求中的单个消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// KIMIResponse 定义了OpenAI聊天接口的响应结构
type KIMIResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Object  string   `json:"object"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Created int64    `json:"created"`
}

// Choice 定义了响应中的choices字段的结构
type Choice struct {
	Index   int `json:"index"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

// Usage 定义了响应中的usage字段的结构
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletions 封装了向OpenAI的/v1/chat/completions接口发送请求的逻辑
func (e *KimiEngine) ChatCompletions(prompt string) (string, error) {
	// 你的OpenAI API密钥
	apiKey := e.apiKey

	// 构建请求体
	request := &ChatRequest{
		Stream:    false,
		UseSearch: true,
		Messages:  []Message{{Content: prompt, Role: "user"}},
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// 构建请求URL
	url := e.baseUrl + "/v1/chat/completions"

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get response from KIMI API: %s", resp.Status)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析响应
	var response KIMIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	// 返回选择的文本
	return response.Choices[0].Message.Content, nil
}
