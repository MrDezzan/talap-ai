package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
	"sync"
	"talkd-backend/models"
)

func openaiClient() (*openai.Client, error) {
	key := os.Getenv("OPENAI_API_KEY")
	url := os.Getenv("LLM_BASE_URL")
	if key == "" {
		return nil, fmt.Errorf("AI_API_KEY not set")
	}
	config := openai.DefaultConfig(key)
	if url != "" {
		config.BaseURL = url
	}
	return openai.NewClientWithConfig(config), nil
}

func llmModel() string {
	m := os.Getenv("LLM_MODEL")
	if m == "" {
		return "anthropic/claude-sonnet-4.6"
	}
	return m
}

type RoadmapStep struct {
	Step        string `json:"step"`
	Duration    string `json:"duration"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Recommendation struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	MatchPercentage int    `json:"match_percentage"`
}

type AnalysisResult struct {
	TopProf     string           `json:"top_profession"`
	Summary     string           `json:"summary"`
	Roadmap     []RoadmapStep    `json:"roadmap"`
	Professions []Recommendation `json:"professions"`
	Grants      []Recommendation `json:"grants"`
}

func AnalyzeProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}

		var input struct {
			Interests   []string `json:"interests"`
			QuizAnswers any      `json:"quiz_answers"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			// Ignore bind errors, try to read from database if empty
		}

		var portfolio models.Portfolio
		db.FirstOrCreate(&portfolio, models.Portfolio{UserID: uid})

		if len(input.Interests) > 0 {
			portfolio.Interests = input.Interests
		}
		if input.QuizAnswers != nil {
			qaJSON, _ := json.Marshal(input.QuizAnswers)
			portfolio.QuizAnswers = qaJSON
		}

		// Extract subject from answers
		var answers map[string]any
		if input.QuizAnswers != nil {
			if m, ok := input.QuizAnswers.(map[string]any); ok {
				answers = m
			}
		} else if len(portfolio.QuizAnswers) > 0 {
			_ = json.Unmarshal(portfolio.QuizAnswers, &answers)
		}

		subject := ""
		workType := ""
		environment := ""
		if answers != nil {
			if s, ok := answers["subject"].(string); ok {
				subject = s
			}
			if w, ok := answers["work_type"].(string); ok {
				workType = w
			}
			if e, ok := answers["environment"].(string); ok {
				environment = e
			}
		}

		// Fallback to interests if quiz is empty
		if subject == "" && len(portfolio.Interests) > 0 {
			firstInterest := strings.ToLower(portfolio.Interests[0])
			if strings.Contains(firstInterest, "программирование") || strings.Contains(firstInterest, "cs") || strings.Contains(firstInterest, "math") || strings.Contains(firstInterest, "информатика") || strings.Contains(firstInterest, "математика") {
				subject = "math"
			} else if strings.Contains(firstInterest, "дизайн") || strings.Contains(firstInterest, "design") || strings.Contains(firstInterest, "рисование") || strings.Contains(firstInterest, "art") {
				subject = "art"
			} else if strings.Contains(firstInterest, "биология") || strings.Contains(firstInterest, "science") || strings.Contains(firstInterest, "физика") || strings.Contains(firstInterest, "химия") {
				subject = "science"
			} else {
				subject = "human"
			}
		}
		if subject == "" {
			subject = "math" // Default fallback
		}

		var result AnalysisResult

		switch subject {
		case "math":
			if workType == "create" || environment == "remote" {
				result = AnalysisResult{
					TopProf: "Software Engineer",
					Summary: "Отличный выбор! Твой аналитический склад ума в сочетании с желанием создавать новые продукты идеально подходит для сферы разработки ПО. Рынок IT сейчас активно растет, особенно в Казахстане.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Изучение базового программирования", Description: "Выбери Python или Go, пройди базовый интерактивный курс по синтаксису и напиши свое первое консольное приложение."},
						{Step: "2", Duration: "3 месяца", Title: "Изучение баз данных и веб-технологий", Description: "Разберись с SQL, веб-фреймворками (например, Gin для Go или FastAPI для Python) и принципами работы сетей."},
						{Step: "3", Duration: "6 месяцев", Title: "Проектное портфолио и стажировка", Description: "Создай 2-3 веб-сервиса на GitHub, начни готовиться к техническим собеседованиям и подавай заявки на стажировки."},
					},
					Professions: []Recommendation{
						{Name: "Software Engineer", Description: "Разработка бэкенда и архитектуры приложений.", MatchPercentage: 95},
						{Name: "AI Specialist", Description: "Создание моделей машинного обучения.", MatchPercentage: 88},
						{Name: "Cybersecurity Analyst", Description: "Обеспечение безопасности данных.", MatchPercentage: 80},
					},
					Grants: []Recommendation{
						{Name: "AITU Talent Grant", Description: "Специальный грант для IT-специалистов в Astana IT University.", MatchPercentage: 92},
						{Name: "Google Step Internship", Description: "Стажировка в Google для студентов младших курсов.", MatchPercentage: 88},
						{Name: "NU Foundation", Description: "Полный грант на обучение в Назарбаев Университете.", MatchPercentage: 75},
					},
				}
			} else if workType == "lead" {
				result = AnalysisResult{
					TopProf: "Product Manager",
					Summary: "Твоя склонность к лидерству и отличные аналитические навыки делают тебя прекрасным кандидатом для управления продуктами. Выступай связующим звеном между разработчиками, дизайнерами и бизнесом.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Основы продуктового мышления", Description: "Изучи жизненный цикл разработки IT-продуктов, базовые бизнес-модели и основы юнит-экономики."},
						{Step: "2", Duration: "3 месяца", Title: "Метрики и аналитика", Description: "Пойми, как работают Amplitude, Google Analytics, и научись принимать решения на основе данных."},
						{Step: "3", Duration: "6 месяцев", Title: "Создание пет-проекта", Description: "Разработай и запусти простой no-code продукт (например, в Glide или Bubble), чтобы показать навыки на практике."},
					},
					Professions: []Recommendation{
						{Name: "Product Manager", Description: "Управление развитием продукта от идеи до запуска.", MatchPercentage: 93},
						{Name: "Data Scientist", Description: "Глубокий анализ бизнес-метрик и данных.", MatchPercentage: 82},
						{Name: "Software Engineer", Description: "Разработка технической базы проекта.", MatchPercentage: 75},
					},
					Grants: []Recommendation{
						{Name: "Suleyman Demirel University Grant", Description: "Бизнес-грант на обучение в SDU.", MatchPercentage: 91},
						{Name: "Болашак", Description: "Престижная международная стипендия.", MatchPercentage: 85},
						{Name: "Turkiye Burslari", Description: "Государственный грант на обучение в Турции.", MatchPercentage: 80},
					},
				}
			} else {
				result = AnalysisResult{
					TopProf: "Data Scientist",
					Summary: "Твоя склонность к логическому анализу и работе со сложными системами делает тебя сильным кандидатом для Data Science и ML. Анализируй тренды, строй прогнозы и находи неочевидные инсайты.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Математика и базовый Python", Description: "Подтяни основы теории вероятностей, линейной алгебры и научись работать с Jupyter Notebook."},
						{Step: "2", Duration: "3 месяца", Title: "Библиотеки анализа данных", Description: "Изучи Pandas, NumPy, SQL для работы с БД и основы визуализации (Matplotlib, Seaborn)."},
						{Step: "3", Duration: "6 месяцев", Title: "Машинное обучение", Description: "Разберись в классических алгоритмах регрессии, классификации и кластеризации с помощью Scikit-Learn."},
					},
					Professions: []Recommendation{
						{Name: "Data Scientist", Description: "Анализ данных и построение ML-моделей.", MatchPercentage: 94},
						{Name: "Cybersecurity Analyst", Description: "Защита информации и выявление угроз.", MatchPercentage: 87},
						{Name: "Software Engineer", Description: "Разработка надежных высоконагруженных сервисов.", MatchPercentage: 78},
					},
					Grants: []Recommendation{
						{Name: "NU Foundation", Description: "Исследовательские гранты в Назарбаев Университете.", MatchPercentage: 90},
						{Name: "AITU Talent Grant", Description: "Обучение Data Science в Astana IT University.", MatchPercentage: 85},
						{Name: "Болашак", Description: "Обучение в топовых зарубежных вузах по техническим специальностям.", MatchPercentage: 78},
					},
				}
			}
		case "art":
			result = AnalysisResult{
				TopProf: "UX/UI Designer",
				Summary: "Творческий потенциал и стремление делать вещи красивыми и удобными — это основа профессии дизайнера интерфейсов. Ты сможешь влиять на то, как миллионы людей используют приложения каждый день.",
				Roadmap: []RoadmapStep{
					{Step: "1", Duration: "1 месяц", Title: "Основы дизайна и Figma", Description: "Изучи интерфейс Figma, принципы композиции, типографики и теории цвета."},
					{Step: "2", Duration: "3 месяца", Title: "Пользовательский опыт (UX)", Description: "Пойми, как проводить исследования пользователей, проектировать карты переходов (CJM) и создавать прототипы."},
					{Step: "3", Duration: "6 месяцев", Title: "Портфолио на Behance/Dribbble", Description: "Оформи 3 красивых кейса (мобильное приложение, веб-сайт) и начни брать первые фриланс-заказы."},
				},
				Professions: []Recommendation{
					{Name: "UX/UI Designer", Description: "Создание удобных цифровых интерфейсов.", MatchPercentage: 96},
					{Name: "Marketing Director", Description: "Разработка визуальных кампаний бренда.", MatchPercentage: 84},
					{Name: "Product Manager", Description: "Управление дизайном и концептом продукта.", MatchPercentage: 75},
				},
				Grants: []Recommendation{
					{Name: "Suleyman Demirel University Grant", Description: "Внутренний грант SDU на гуманитарные и медиа направления.", MatchPercentage: 88},
					{Name: "Болашак", Description: "Стипендия для обучения дизайну за рубежом.", MatchPercentage: 80},
					{Name: "Turkiye Burslari", Description: "Гранты в ведущих вузах искусств Турции.", MatchPercentage: 75},
				},
			}
		case "science":
			result = AnalysisResult{
				TopProf: "Medical Doctor",
				Summary: "Стремление приносить реальную пользу обществу через научные или практические дисциплины — очень благородный выбор. Медицина и классическая инженерия остаются столпами стабильной карьеры.",
				Roadmap: []RoadmapStep{
					{Step: "1", Duration: "1 месяц", Title: "Углубленная подготовка по биологии и химии", Description: "Сфокусируйся на ключевых темах общей биологии, анатомии человека и органической химии."},
					{Step: "2", Duration: "3 месяца", Title: "Волонтерство в медицинских организациях", Description: "Попробуй устроиться волонтером в больницу или научную лабораторию, чтобы понять профессию изнутри."},
					{Step: "3", Duration: "6 месяцев", Title: "Подготовка к конкурсу грантов", Description: "Сдай ЕНТ на максимальный балл или подготовь сильное резюме для получения международных стипендий."},
				},
				Professions: []Recommendation{
					{Name: "Medical Doctor", Description: "Диагностика и лечение пациентов.", MatchPercentage: 94},
					{Name: "Civil Engineer", Description: "Проектирование безопасных объектов.", MatchPercentage: 85},
					{Name: "Data Scientist", Description: "Аналитика в сфере биоинформатики.", MatchPercentage: 70},
				},
				Grants: []Recommendation{
					{Name: "NU Foundation", Description: "Медицинская школа Назарбаев Университета.", MatchPercentage: 93},
					{Name: "Болашак", Description: "Гранты на обучение в лучших медицинских резидентурах мира.", MatchPercentage: 88},
					{Name: "Turkiye Burslari", Description: "Обучение в престижных турецких медицинских вузах.", MatchPercentage: 82},
				},
			}
		default: // "human" или любой другой случай
			result = AnalysisResult{
				TopProf: "Marketing Director",
				Summary: "Отличные коммуникативные навыки и способность координировать процессы делают тебя прирожденным лидером или стратегом. Сфера продакт-менеджмента и маркетинга сейчас крайне востребована.",
				Roadmap: []RoadmapStep{
					{Step: "1", Duration: "1 месяц", Title: "Основы маркетинга и психологии потребителя", Description: "Пойми принципы позиционирования, сегментации рынка и построения бренда."},
					{Step: "2", Duration: "3 месяца", Title: "Цифровой маркетинг и таргет", Description: "Изучи инструменты таргетированной рекламы, SEO, контекстную рекламу и работу с инфлюенсерами."},
					{Step: "3", Duration: "6 месяцев", Title: "Первые кейсы и продвижение", Description: "Помоги местному бизнесу или стартапу настроить поток клиентов и оформи этот кейс в портфолио."},
				},
				Professions: []Recommendation{
					{Name: "Marketing Director", Description: "Разработка стратегии продвижения и рекламы бренда.", MatchPercentage: 93},
					{Name: "Product Manager", Description: "Создание и развитие продуктов под запросы рынка.", MatchPercentage: 88},
					{Name: "UX/UI Designer", Description: "Понимание психологии пользователей для улучшения интерфейсов.", MatchPercentage: 72},
				},
				Grants: []Recommendation{
					{Name: "Suleyman Demirel University Grant", Description: "Гранты в SDU на маркетинг и бизнес-управление.", MatchPercentage: 91},
					{Name: "Болашак", Description: "Обучение бизнес-маркетингу и менеджменту за рубежом.", MatchPercentage: 85},
					{Name: "Turkiye Burslari", Description: "Гранты в ведущих бизнес-школах Турции.", MatchPercentage: 80},
				},
			}
		}

		db.FirstOrCreate(&portfolio, models.Portfolio{UserID: uid})
		resJSON, _ := json.Marshal(result)
		db.Model(&portfolio).Updates(models.Portfolio{
			AIResult:  resJSON,
			UpdatedAt: time.Now(),
		})

		c.JSON(http.StatusOK, result)
	}
}

var (
	lastRequestTime = make(map[uint]time.Time)
	rateLimitMutex  = &sync.Mutex{}
)

func AIChat(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			ThreadID uint   `json:"thread_id"`
			Message  string `json:"message"`
			UserInfo struct {
				Name    string `json:"name"`
				Grade   string `json:"grade"`
				City    string `json:"city"`
				Summary string `json:"summary"`
				TopProf string `json:"top_profession"`
			} `json:"user_info"`
			History []map[string]string `json:"history"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			log.Printf("AIChat Bind Error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}

		rateLimitMutex.Lock()
		last, ok := lastRequestTime[uid]
		if ok && time.Since(last) < 2*time.Second {
			rateLimitMutex.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Пожалуйста, подождите немного перед следующим сообщением"})
			return
		}
		lastRequestTime[uid] = time.Now()
		rateLimitMutex.Unlock()

		var thread models.ChatThread
		if input.ThreadID != 0 {
			db.Where("id = ? AND user_id = ?", input.ThreadID, uid).First(&thread)
		}
		if thread.ID == 0 {
			thread = models.ChatThread{
				UserID:    uid,
				Title:     input.Message[:min(30, len(input.Message))],
				UpdatedAt: time.Now(),
			}
			db.Create(&thread)
		}

		db.Create(&models.ChatMessage{
			ThreadID: thread.ID,
			Role:     "user",
			Content:  input.Message,
		})

		var portfolio models.Portfolio
		db.Where("user_id = ?", uid).First(&portfolio)

		achStr := "[]"
		if len(portfolio.Achievements) > 0 {
			achStr = string(portfolio.Achievements)
		}

		portfolioStatus := fmt.Sprintf(`
БИО: %s
ЕНТ: %s
АНГЛИЙСКИЙ: %s
НАВЫКИ: %v
ДОСТИЖЕНИЯ: %s`, 
		portfolio.Bio, portfolio.ENT, portfolio.English, portfolio.Skills, achStr)

		currentDate := time.Now().Format("02 January 2006")
		systemPrompt := fmt.Sprintf(`Ты — экспертный ИИ-советник Talap. 
СЕГОДНЯ: %s.
ЯЗЫК: Отвечай СТРОГО на том же языке, на котором пишет пользователь (RU, KZ или EN).

ДАННЫЕ ПОЛЬЗОВАТЕЛЯ:
- Имя: %s
- Класс: %s
- Город: %s
- Анализ профиля: %s
- Текущая профессия-цель: %s

ПОЛНОЕ ПОРТФОЛИО ИЗ БАЗЫ:
%s

ЦЕЛЬ: Помочь пользователю в карьере, образовании и поиске грантов.
ОСОБАЯ ЗАДАЧА: Автоматическое составление идеального CV (резюме).

ЭТАЛОН СТРУКТУРЫ CV (на основе шаблона Vlad Mishustin):
1. Заголовок: ФИО | Телефон | Email | Ссылки (LinkedIn/GitHub/Портфолио)
2. Tagline: Краткий экспертный заголовок (например: "Junior Frontend Developer | React | 3+ Проекта")
3. PROFESSIONAL EXPERIENCE / PROJECTS:
   - Название компании или Проекта | Роль | Даты
   - Краткое описание компании/проекта
   - Достижения (ОБЯЗАТЕЛЬНО: использование цифр и глаголов действия, например: "Увеличил скорость загрузки на 40%%", "Разработал систему с нуля")
   - Stack: Технологии
4. ACHIEVEMENTS: Хакатоны, олимпиады, призы, каналы, выступления.
5. EDUCATION: ВУЗ/Школа, специальность, даты.
6. LANGUAGES & HOBBIES.

ТВОЯ ЛОГИКА СБОРА ИНФОРМАЦИИ:
1. Проверь, достаточно ли у тебя данных для составления CV. Тебе нужно:
   - Контакты (телефон, ссылки)
   - Опыт (минимум 1-2 проекта или места работы с деталями)
   - Достижения (что-то конкретное, кроме учебы)
   - Навыки (стек технологий)
2. Если данных НЕТ в портфолио, ТЫ ДОЛЖЕН вежливо и профессионально спросить их в ходе беседы.
3. Если данных ДОСТАТОЧНО (или пользователь просит CV), ты ОБЯЗАН заполнить объект "cv_data" в ответе.
4. Если ты видишь, что пользователь рассказал о новом проекте или навыке — сразу обнови "cv_data" и предложи скачать обновленное резюме.

КРИТИЧЕСКИЕ ПРАВИЛА:
1. Используй всю информацию из ПОРТФОЛИО.
2. НИКОГДА не спрашивай то, что уже известно.
3. Текст CV должен быть ВПЕЧАТЛЯЮЩИМ (как у Vlad Mishustin) — используй сильные глаголы (Designed, Launched, Optimized, Migrated).
4. Каждому шагу в "roadmap" ОБЯЗАТЕЛЬНО давай осмысленное название в "title" и экспертное описание в "description". Не оставляй их пустыми.
5. Возвращай "cv_data" ТОЛЬКО когда есть реальное наполнение.

ОТВЕЧАЙ СТРОГО В JSON: 
{
  "message": "...", 
  "roadmap": [
    {
      "title": "Название этапа",
      "description": "Что именно нужно сделать и изучить",
      "duration": "Срок (например, 2 месяца)"
    }
  ],
  "cv_data": {
    "name": "...",
    "contact": "Phone | Email | Links",
    "tagline": "...",
    "summary": "...",
    "experience": [
      {
        "company": "...",
        "role": "...",
        "period": "...",
        "description": "...",
        "achievements": ["...", "..."],
        "stack": "..."
      }
    ],
    "education": [{"school": "...", "degree": "...", "period": "..."}],
    "skills": ["...", "..."],
    "achievements_list": ["...", "..."],
    "languages": "...",
    "hobbies": "..."
  } или null
}`, 
		currentDate, input.UserInfo.Name, input.UserInfo.Grade, input.UserInfo.City, input.UserInfo.Summary, input.UserInfo.TopProf, portfolioStatus)

		msgs := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		}
		for _, h := range input.History {
			role := openai.ChatMessageRoleUser
			if h["role"] == "assistant" { role = openai.ChatMessageRoleAssistant }
			msgs = append(msgs, openai.ChatCompletionMessage{Role: role, Content: h["content"]})
		}
		msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: input.Message})

		client, err := openaiClient()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    llmModel(),
			Messages: msgs,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI failed: " + err.Error()})
			return
		}

		raw := strings.TrimSpace(resp.Choices[0].Message.Content)
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)

		var result struct {
			Message string `json:"message"`
			Roadmap any    `json:"roadmap"`
			CVData  any    `json:"cv_data"`
		}
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			log.Printf("AI response not valid JSON, using raw text. Error: %v", err)
			result.Message = raw
		}
		if result.Message == "" {
			result.Message = raw
		}

		rmJSON, _ := json.Marshal(result.Roadmap)
		cvJSON, _ := json.Marshal(result.CVData)
		db.Create(&models.ChatMessage{
			ThreadID: thread.ID,
			Role:     "assistant",
			Content:  result.Message,
			Roadmap:  rmJSON,
			CVData:   cvJSON,
		})
		db.Model(&thread).Update("updated_at", time.Now())

		c.JSON(http.StatusOK, gin.H{
			"thread_id": thread.ID,
			"message":   result.Message,
			"roadmap":   result.Roadmap,
			"cv_data":   result.CVData,
		})
	}
}

func GetChatThreads(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}
		threads := []models.ChatThread{}
		db.Where("user_id = ?", uid).Order("updated_at desc").Find(&threads)
		c.JSON(http.StatusOK, threads)
	}
}

func GetChatMessages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}
		threadID := c.Param("id")
		
		var thread models.ChatThread
		if err := db.Where("id = ? AND user_id = ?", threadID, uid).First(&thread).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Thread not found"})
			return
		}

		messages := []models.ChatMessage{}
		db.Where("thread_id = ?", threadID).Order("created_at asc").Find(&messages)
		c.JSON(http.StatusOK, messages)
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
