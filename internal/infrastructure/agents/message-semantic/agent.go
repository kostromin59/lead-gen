package messagesemantic

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/achetronic/adk-utils-go/genai/openai/completions"
	"github.com/google/uuid"
	"github.com/kostromin59/lead-gen/internal/models"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

type Agent struct {
	runner  *runner.Runner
	session session.Service
}

func New() *Agent {
	llmModel := completions.New(completions.Config{
		BaseURL:   "http://192.168.1.100:11434/v1",
		ModelName: "gemma4:e4b",
	})

	itemSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"message_id": {
				Type:        genai.TypeString,
				Description: "ID сообщения из входных данных",
			},
			"domain": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "Предметная область сообщения",
			},
			"entities": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "Конкретика, детали (города, бренды, модели и т.д.)",
			},
			"intent": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "Намерение сообщения",
			},
		},
		Required:    []string{"message_id", "domain", "entities", "intent"},
		Description: "Теги для одного сообщения",
	}

	// Схема всего ответа - массив объектов
	schema := &genai.Schema{
		Type:        genai.TypeArray,
		Items:       itemSchema,
		Description: "Массив объектов с тегами для каждого сообщения",
	}

	agent, err := llmagent.New(llmagent.Config{
		Name:         "message-semantic",
		Model:        llmModel,
		OutputSchema: schema,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: false,
				ThinkingBudget:  new(int32(0)),
				ThinkingLevel:   genai.ThinkingLevelLow,
			},
			MaxOutputTokens: 32000,
		},

		Description: "Агент, который будет определять теги для сообщений, Domain (предметная область), Entities (детали, конкретика), Intent (намерение)",
		Instruction: `Ты — агент, что анализирует сообщения из мессенджера.
		Будут даны некоторое количество сообщений из одного чата.
		К этим сообщениям будет приложено название чата и его описание для более точного анализа.
		Твоя цель для каждого сообщения по его ID, содержанию, времени, названию и описанию, чата указать следующие теги: domain (предметная область), entities (детали, конкретика), intent (намерение).
		В каждом теге может быть несколько значений. Выводы нужно делать на основе всех сообщений (истории), что тебе были переданы для более точного анализа.
		В domain указывается общая предметная область: развлечения, автомобили, IT, медицина, новости, техника, путешествия и так далее, что отвечают общей предметной области. Я привёл лишь несколько примеров из многих вариантов.
		В entities указывается конкретика: название города, бренда, марка, место, модель и любые другие, что отвечают деталям. Я привёл лишь несколько примеров из многих вариантов.
		В intent указывается намерение: покупка, продажа, вопрос, обсуждение, шутка и любые другие, что позволит определить намерение или характер сообщения. Я привёл лишь несколько примеров из многих вариантов.
		Необходимо учитывать, что в рамках истории сообщений, может быть несколько разных предметных областей, конкретики и намерений.
		Запрещается выдумывать любую информацию. Если не знаешь — нужно указать, что пусто. Нужно указать теги для всех сообщений. 
		Если передано 100 сообщений, то необходимо в ответе указать теги для 100 сообщений (в том числе пустые).
		Теги по возможности нужно писать на русском языке, если это не название собственное, деталь или конкретика. В message_id всегда подставляй message_id, не выдумывай его. Если значений нет, значит отдавай пустой массив.`,
	})
	if err != nil {
		panic(err)
	}

	s := session.InMemoryService()

	r, err := runner.New(runner.Config{
		AppName:        "message-semantic",
		Agent:          agent,
		SessionService: s,
	})
	if err != nil {
		panic(err)
	}

	return &Agent{
		runner:  r,
		session: s,
	}
}

func (a *Agent) Handle(ctx context.Context, chatName, chatDescription string, messages []models.Message) (map[string]models.Semantic, error) {
	type inputMsg struct {
		SenderID    string    `json:"sender_id"`
		MessageID   string    `json:"message_id"`
		ThreadID    *string   `json:"thread_id,omitempty"`
		Content     string    `json:"content"`
		MessageTime time.Time `json:"message_time"`
		CreatedAt   time.Time `json:"created_at"`
	}

	type input struct {
		ChatName        string `json:"chat_name"`
		ChatDescription string `json:"chat_description"`
		Messages        []inputMsg
	}

	msgs := []inputMsg{}

	for _, m := range messages {
		msgs = append(msgs, inputMsg{
			MessageID:   m.MessageID,
			ThreadID:    m.ThreadID,
			Content:     m.Content,
			MessageTime: m.MessageTime,
			SenderID:    m.SenderID,
		})
	}

	i := input{
		ChatName:        chatName,
		ChatDescription: chatDescription,
		Messages:        msgs,
	}
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	uid := uuid.NewString()
	sid := uuid.NewString()
	if _, err := a.session.Create(ctx, &session.CreateRequest{
		AppName:   "message-semantic",
		State:     map[string]any{},
		UserID:    uid,
		SessionID: sid,
	}); err != nil {
		return nil, err
	}

	content := genai.NewContentFromText(string(b), genai.RoleUser)
	events := a.runner.Run(ctx, uid, sid, content, agent.RunConfig{
		// StreamingMode: agent.StreamingModeSSE,
	})
	log.Println("events")
	res := ""
	for event, err := range events {
		log.Println("new event")
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		if event.Content == nil {
			continue
		}

		for _, part := range event.Content.Parts {
			if part.Thought {
				log.Printf("thought: %q", part.Text)
				continue
			}
			if part.Text != "" {
				res += part.Text
				log.Printf("res: %q", res)
			}
		}
	}

	type response struct {
		Domain    []string `json:"domain"`
		Entities  []string `json:"entities"`
		Intent    []string `json:"intent"`
		MessageID string   `json:"message_id"`
	}

	var resp []response
	if err := json.Unmarshal([]byte(res), &resp); err != nil {
		return nil, err
	}

	result := map[string]models.Semantic{}
	for _, r := range resp {
		result[r.MessageID] = models.Semantic{
			Domain:   r.Domain,
			Entities: r.Entities,
			Intent:   r.Intent,
		}
	}

	return result, nil
}
