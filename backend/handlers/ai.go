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
	if key == "" {
		key = os.Getenv("OPENROUTER_API_KEY")
	}
	url := os.Getenv("LLM_BASE_URL")
	if key == "" {
		return nil, fmt.Errorf("neither OPENAI_API_KEY nor OPENROUTER_API_KEY is set")
	}
	config := openai.DefaultConfig(key)
	if url != "" {
		config.BaseURL = url
	} else {
		config.BaseURL = "https://openrouter.ai/api/v1"
	}
	return openai.NewClientWithConfig(config), nil
}

func llmModel() string {
	m := os.Getenv("LLM_MODEL")
	if m == "" {
		return "google/gemini-3.1-flash"
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

		// Quiz variables
		subject := ""
		workType := ""
		environment := ""
		goal := ""
		problemSolving := ""
		leisure := ""
		projectRole := ""

		if answers != nil {
			if s, ok := answers["subject"].(string); ok { subject = s }
			if w, ok := answers["work_type"].(string); ok { workType = w }
			if e, ok := answers["environment"].(string); ok { environment = e }
			if g, ok := answers["goal"].(string); ok { goal = g }
			if p, ok := answers["problem_solving"].(string); ok { problemSolving = p }
			if l, ok := answers["leisure"].(string); ok { leisure = l }
			if r, ok := answers["project_role"].(string); ok { projectRole = r }
		}

		// Scoring categories
		itScore := 0
		designScore := 0
		businessScore := 0
		medicineScore := 0
		engineeringScore := 0

		// 1. Subject scoring
		switch subject {
		case "math":
			itScore += 20
			engineeringScore += 15
			businessScore += 5
		case "art":
			designScore += 20
		case "human":
			businessScore += 20
			designScore += 5
		case "science":
			medicineScore += 20
			engineeringScore += 15
		}

		// 2. Work type scoring
		switch workType {
		case "team":
			businessScore += 10
			medicineScore += 10
		case "solo":
			itScore += 10
			engineeringScore += 10
		case "lead":
			businessScore += 15
		case "create":
			itScore += 10
			designScore += 15
			engineeringScore += 10
		}

		// 3. Environment scoring
		switch environment {
		case "office":
			businessScore += 10
			itScore += 5
		case "remote":
			itScore += 15
			designScore += 15
		case "travel":
			businessScore += 15
		case "lab":
			medicineScore += 15
			engineeringScore += 15
		}

		// 4. Goal scoring
		switch goal {
		case "money":
			itScore += 10
			businessScore += 15
		case "impact":
			medicineScore += 15
			engineeringScore += 5
		case "fame":
			designScore += 15
			businessScore += 5
		case "growth":
			itScore += 10
			engineeringScore += 10
			medicineScore += 5
		}

		// 5. Problem solving scoring
		switch problemSolving {
		case "logic":
			itScore += 10
			engineeringScore += 10
			medicineScore += 5
		case "intuition":
			designScore += 15
		case "people":
			businessScore += 10
			medicineScore += 10
		case "practical":
			itScore += 5
			engineeringScore += 10
			designScore += 5
		}

		// 6. Leisure scoring
		switch leisure {
		case "gadgets":
			itScore += 15
			engineeringScore += 15
		case "draw":
			designScore += 20
		case "books":
			medicineScore += 15
			engineeringScore += 10
		case "games":
			businessScore += 15
			designScore += 5
		}

		// 7. Project role scoring
		switch projectRole {
		case "builder":
			itScore += 15
			engineeringScore += 15
			designScore += 5
		case "presenter":
			businessScore += 15
			designScore += 10
		case "manager":
			businessScore += 20
		case "researcher":
			medicineScore += 20
			engineeringScore += 10
		}

		// Find the highest scoring category
		maxScore := itScore
		category := "it"

		if designScore > maxScore {
			maxScore = designScore
			category = "design"
		}
		if businessScore > maxScore {
			maxScore = businessScore
			category = "business"
		}
		if medicineScore > maxScore {
			maxScore = medicineScore
			category = "medicine"
		}
		if engineeringScore > maxScore {
			maxScore = engineeringScore
			category = "engineering"
		}

		var result AnalysisResult

		switch category {
		case "it":
			if leisure == "gadgets" && projectRole == "builder" {
				result = AnalysisResult{
					TopProf: "AI/ML Specialist",
					Summary: "Отличный выбор! Твоя страсть к инновациям и желание создавать интеллектуальные алгоритмы идеально подходят для сферы ИИ. Профессия разработчика машинного обучения имеет невероятную скорость роста спроса.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1-2 месяца", Title: "Математический фундамент и Python", Description: "Изучи Python на продвинутом уровне, а также освежи знания линейной алгебры и теории вероятностей."},
						{Step: "2", Duration: "3 месяца", Title: "Основы Data Science и ML", Description: "Научись работать с Jupyter Notebook, библиотеками Pandas, NumPy и классическими алгоритмами Scikit-Learn."},
						{Step: "3", Duration: "6 месяцев", Title: "Глубокое обучение и Нейросети", Description: "Переходи к фреймворкам PyTorch или TensorFlow. Создай проект по распознаванию образов или анализу текста для портфолио."},
					},
					Professions: []Recommendation{
						{Name: "AI/ML Specialist", Description: "Разработка архитектур нейросетей и генеративного ИИ.", MatchPercentage: 96},
						{Name: "Data Scientist", Description: "Анализ больших данных и прогнозирование трендов.", MatchPercentage: 88},
						{Name: "Software Engineer", Description: "Проектирование крупных бэкенд-систем.", MatchPercentage: 82},
					},
					Grants: []Recommendation{
						{Name: "AITU Talent Grant", Description: "Специальный IT-грант в Astana IT University.", MatchPercentage: 94},
						{Name: "Google STEP Internship", Description: "Оплачиваемая летняя IT-стажировка в Google.", MatchPercentage: 90},
						{Name: "KBTU Olympiad Grant", Description: "100% покрытие обучения в КБТУ на IT-направлениях.", MatchPercentage: 85},
					},
				}
			} else if environment == "remote" {
				result = AnalysisResult{
					TopProf: "Software Engineer",
					Summary: "Твой аналитический склад ума в сочетании со стремлением работать самостоятельно отлично подходит для разработки ПО. Разработка бэкенда дает полную свободу в выборе места работы.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Базовые алгоритмы и синтаксис", Description: "Освой основы программирования на Go или Python, разберись с типами данных и простыми алгоритмами."},
						{Step: "2", Duration: "3 месяца", Title: "Базы данных и Архитектура веб-приложений", Description: "Изучи SQL, проектирование реляционных баз данных и разработку REST API на современных фреймворках."},
						{Step: "3", Duration: "6 месяцев", Title: "Пет-проекты и GitHub", Description: "Сделай полноценный проект (например, API для интернет-магазина), залей на GitHub и начни подавать резюме."},
					},
					Professions: []Recommendation{
						{Name: "Software Engineer", Description: "Разработка архитектур приложений и бэкенд-сервисов.", MatchPercentage: 95},
						{Name: "Mobile Developer", Description: "Разработка приложений для iOS и Android.", MatchPercentage: 87},
						{Name: "Cybersecurity Analyst", Description: "Анализ систем безопасности и сетей.", MatchPercentage: 78},
					},
					Grants: []Recommendation{
						{Name: "AITU Talent Grant", Description: "IT-грант для абитуриентов в Astana IT University.", MatchPercentage: 92},
						{Name: "Google STEP Internship", Description: "Программа стажировки и стипендий от Google.", MatchPercentage: 88},
						{Name: "Грант Назарбаев Университета (NU)", Description: "Англоязычные программы бакалавриата в NU.", MatchPercentage: 80},
					},
				}
			} else if problemSolving == "logic" {
				result = AnalysisResult{
					TopProf: "Cybersecurity Analyst",
					Summary: "Логика и внимание к деталям — главные качества белого хакера. Ты научишься мыслить как злоумышленник, чтобы выстраивать надежную защиту корпоративных систем от угроз.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Основы сетей и Linux", Description: "Освой администрирование Linux, пойми работу стека сетевых протоколов TCP/IP."},
						{Step: "2", Duration: "3 месяца", Title: "Анализ уязвимостей", Description: "Изучи основы криптографии, а также инструменты сканирования сетей (Nmap, Wireshark, Burp Suite)."},
						{Step: "3", Duration: "6 месяцев", Title: "Практика на CTF", Description: "Решай задания по безопасности на платформах HackTheBox и TryHackMe для формирования практического опыта."},
					},
					Professions: []Recommendation{
						{Name: "Cybersecurity Analyst", Description: "Защита информационных систем и предотвращение взломов.", MatchPercentage: 93},
						{Name: "Software Engineer", Description: "Разработка защищенного кода.", MatchPercentage: 81},
						{Name: "Data Scientist", Description: "Анализ аномалий в поведении сетевого трафика.", MatchPercentage: 75},
					},
					Grants: []Recommendation{
						{Name: "KBTU Olympiad Grant", Description: "Бесплатное обучение на факультете кибербезопасности КБТУ.", MatchPercentage: 90},
						{Name: "AITU Talent Grant", Description: "Гранты в сфере компьютерной безопасности AITU.", MatchPercentage: 85},
						{Name: "Болашак", Description: "Обучение кибербезопасности в ведущих вузах США и Европы.", MatchPercentage: 82},
					},
				}
			} else {
				result = AnalysisResult{
					TopProf: "Mobile Developer",
					Summary: "Каждый использует мобильные приложения ежедневно. Твоя техническая подготовка позволит тебе создавать быстрые, удобные и красивые приложения для iOS или Android.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Выбор платформы и языка", Description: "Выбери Swift для iOS или Kotlin для Android. Пройди базовый курс по объектно-ориентированному программированию."},
						{Step: "2", Duration: "3 месяца", Title: "Проектирование UI и работа с API", Description: "Изучи основы верстки экранов (SwiftUI или Jetpack Compose) и интеграцию с бэкендом через REST API."},
						{Step: "3", Duration: "6 месяцев", Title: "Публикация первого приложения", Description: "Создай и опубликуй свое первое простое приложение в App Store или Google Play, чтобы доказать свои навыки работодателям."},
					},
					Professions: []Recommendation{
						{Name: "Mobile Developer", Description: "Создание приложений для смартфонов и планшетов.", MatchPercentage: 94},
						{Name: "Software Engineer", Description: "Веб-разработка и создание API для приложений.", MatchPercentage: 85},
						{Name: "UX/UI Designer", Description: "Обеспечение эстетичности интерфейсов мобильных приложений.", MatchPercentage: 78},
					},
					Grants: []Recommendation{
						{Name: "AITU Talent Grant", Description: "Стипендии на мобильную разработку в Astana IT University.", MatchPercentage: 91},
						{Name: "SDU Internal Grant", Description: "Внутренние образовательные гранты SDU.", MatchPercentage: 85},
						{Name: "Google STEP Internship", Description: "Стажировка в европейских командах Google.", MatchPercentage: 79},
					},
				}
			}

		case "design":
			if projectRole == "builder" || problemSolving == "logic" {
				result = AnalysisResult{
					TopProf: "UX/UI Designer",
					Summary: "Идеальный баланс аналитики и творчества! Твоя способность мыслить логически поможет тебе создавать удобную структуру интерфейсов, а эстетический вкус — делать их привлекательными.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Базовые инструменты в Figma", Description: "Освой основы создания компонентов, автолейаутов и стилей в Figma."},
						{Step: "2", Duration: "2 месяца", Title: "Проектирование пользовательского пути (UX)", Description: "Научись проводить коридорные тестирования, строить карты CJM и проектировать логику интерфейсов."},
						{Step: "3", Duration: "5 месяцев", Title: "Реальные кейсы в портфолио", Description: "Переработай дизайн неудобного приложения или создай концепт нового и выложи на Behance."},
					},
					Professions: []Recommendation{
						{Name: "UX/UI Designer", Description: "Проектирование пользовательского опыта и визуального дизайна интерфейсов.", MatchPercentage: 95},
						{Name: "Content Creator / SMM", Description: "Создание визуального стиля для социальных медиа.", MatchPercentage: 80},
						{Name: "Product Manager", Description: "Управление концептом и развитием продукта.", MatchPercentage: 74},
					},
					Grants: []Recommendation{
						{Name: "SDU Internal Grant", Description: "Внутренний грант SDU на медиа и дизайн-направления.", MatchPercentage: 92},
						{Name: "Turkiye Burslari", Description: "Стипендии в лучших вузах искусств Турции.", MatchPercentage: 84},
						{Name: "Болашак", Description: "Обучение зарубежному медиадизайну и UX.", MatchPercentage: 78},
					},
				}
			} else {
				result = AnalysisResult{
					TopProf: "Content Creator / SMM",
					Summary: "Твоя креативность и понимание людей делают тебя отличным контент-стратегом. Ты сможешь легко находить общий язык с аудиторией и создавать вирусный визуальный и текстовый контент.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Основы сторителлинга и мобильной съемки", Description: "Пойми принципы удержания внимания, раскадровки, монтажа видео и цветокоррекции на телефоне."},
						{Step: "2", Duration: "2 месяца", Title: "Алгоритмы платформ и продвижение", Description: "Изучи, как работают алгоритмы рекомендаций в TikTok, Instagram и YouTube, и освой основы SMM-копирайтинга."},
						{Step: "3", Duration: "4 месяца", Title: "Продвижение первых клиентов", Description: "Помоги набрать просмотры местному бренду или начни вести экспертный блог, чтобы показать живую статистику."},
					},
					Professions: []Recommendation{
						{Name: "Content Creator / SMM", Description: "Создание контента и продвижение личного или бизнес-бренда.", MatchPercentage: 94},
						{Name: "UX/UI Designer", Description: "Создание визуального контента и дизайн страниц.", MatchPercentage: 82},
						{Name: "Marketing Director", Description: "Разработка крупных рекламных стратегий.", MatchPercentage: 79},
					},
					Grants: []Recommendation{
						{Name: "SDU Internal Grant", Description: "Скидки до 100% на журналистику и связи с общественностью в SDU.", MatchPercentage: 90},
						{Name: "Turkiye Burslari", Description: "Связи с общественностью и медиапроизводство в Турции.", MatchPercentage: 85},
						{Name: "Болашак", Description: "Гранты на зарубежное обучение в сфере медиакоммуникаций.", MatchPercentage: 75},
					},
				}
			}

		case "business":
			if projectRole == "manager" {
				result = AnalysisResult{
					TopProf: "Product Manager",
					Summary: "Отличные организаторские качества и умение сплотить команду делают тебя идеальным руководителем продуктов. Ты будешь направлять команду разработки к общей цели.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Разбор Agile и Scrum", Description: "Пойми, как организуются процессы разработки в IT-компаниях по гибким методологиям."},
						{Step: "2", Duration: "2 месяца", Title: "Продуктовые метрики и гипотезы", Description: "Разберись, как проверять гипотезы, создавать MVP (минимально жизнеспособный продукт) и анализировать отзывы."},
						{Step: "3", Duration: "6 месяцев", Title: "Запуск своего No-code проекта", Description: "Создай прототип приложения на Glide/Bubble и презентуй его в резюме для получения первой работы."},
					},
					Professions: []Recommendation{
						{Name: "Product Manager", Description: "Управление жизненным циклом продукта и развитие бизнеса.", MatchPercentage: 96},
						{Name: "Marketing Director", Description: "Реклама и вывод новых продуктов на рынок.", MatchPercentage: 85},
						{Name: "Financial Analyst", Description: "Расчет юнит-экономики и бюджетирование.", MatchPercentage: 76},
					},
					Grants: []Recommendation{
						{Name: "SDU Internal Grant", Description: "Гранты в SDU на менеджмент и управление бизнесом.", MatchPercentage: 93},
						{Name: "Болашак", Description: "Престижная бизнес-магистратура MBA за рубежом.", MatchPercentage: 88},
						{Name: "Turkiye Burslari", Description: "Бизнес-образование в ведущих университетах Турции.", MatchPercentage: 80},
					},
				}
			} else if problemSolving == "logic" {
				result = AnalysisResult{
					TopProf: "Financial Analyst",
					Summary: "Сочетание прагматичности, любви к математике и понимания бизнес-процессов — залог успеха в сфере финансов. Ты будешь анализировать инвестиции, риски и выстраивать стратегии доходности.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Углубленный Excel и Финансовый учет", Description: "Изучи формулы, сводные таблицы Excel и разберись в основных отчетах (P&L, Cash Flow, Balance Sheet)."},
						{Step: "2", Duration: "3 месяца", Title: "Финансовое моделирование", Description: "Научись строить финансовые модели компаний и оценивать инвестиционную привлекательность."},
						{Step: "3", Duration: "6 месяцев", Title: "Подготовка к сертификации CFA", Description: "Начни готовиться к базовому уровню CFA и пройди стажировку в банке или консалтинговой компании."},
					},
					Professions: []Recommendation{
						{Name: "Financial Analyst", Description: "Анализ финансовых потоков, рисков и инвестиционная оценка.", MatchPercentage: 94},
						{Name: "Product Manager", Description: "Экономическая оценка эффективности продуктов.", MatchPercentage: 84},
						{Name: "Data Scientist", Description: "Машинное обучение в финансовом секторе.", MatchPercentage: 73},
					},
					Grants: []Recommendation{
						{Name: "SDU Internal Grant", Description: "Скидки за высокие баллы на экономику и финансы в SDU.", MatchPercentage: 91},
						{Name: "Болашак", Description: "Магистратура в лучших бизнес-школах Великобритании и США.", MatchPercentage: 86},
						{Name: "Yessenov Foundation Grant", Description: "Стажировки и гранты для будущих лидеров бизнеса.", MatchPercentage: 80},
					},
				}
			} else {
				result = AnalysisResult{
					TopProf: "Marketing Director",
					Summary: "Стратегический взгляд и глубокое понимание человеческой психологии позволят тебе вести крупные рекламные проекты. Ты будешь определять, как именно аудитория видит продукты и услуги.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Теория маркетинга и позиционирования", Description: "Изучи основы 4P маркетинга, анализ конкурентов и позиционирование бренда."},
						{Step: "2", Duration: "3 месяца", Title: "Performance и Digital-маркетинг", Description: "Освой таргетированную рекламу, контекстную рекламу, веб-аналитику и воронки продаж."},
						{Step: "3", Duration: "6 месяцев", Title: "Создание реальных кейсов", Description: "Помоги привлечь клиентов локальному бизнесу и оформи результаты (ROI, CPC, Лиды) в кейс-резюме."},
					},
					Professions: []Recommendation{
						{Name: "Marketing Director", Description: "Разработка стратегии бренда и управление рекламным бюджетом.", MatchPercentage: 95},
						{Name: "Product Manager", Description: "Анализ потребностей клиентов и создание MVP.", MatchPercentage: 87},
						{Name: "Content Creator / SMM", Description: "Разработка креативного контента для рекламы.", MatchPercentage: 80},
					},
					Grants: []Recommendation{
						{Name: "SDU Internal Grant", Description: "Стипендии SDU на маркетинг и связи с общественностью.", MatchPercentage: 93},
						{Name: "Болашак", Description: "Обучение бренд-менеджменту в лучших университетах мира.", MatchPercentage: 85},
						{Name: "Turkiye Burslari", Description: "Полные государственные гранты на обучение маркетингу в Турции.", MatchPercentage: 79},
					},
				}
			}

		case "medicine":
			if projectRole == "researcher" {
				result = AnalysisResult{
					TopProf: "Biotech Specialist",
					Summary: "Медицина будущего! Твой исследовательский интерес к науке позволит тебе создавать инновации в лабораториях: новые вакцины, методы лечения рака и биотехнологические продукты.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Глубокая молекулярная биология", Description: "Освой основы генетики, клеточной биологии и органической химии."},
						{Step: "2", Duration: "3 месяца", Title: "Лабораторные методы исследований", Description: "Изучи теорию ПЦР, хроматографии и принципы культивирования клеток."},
						{Step: "3", Duration: "6 месяцев", Title: "Участие в научных проектах", Description: "Найди научного руководителя, начни участвовать в реальных лабораторных тестах и готовь научную работу."},
					},
					Professions: []Recommendation{
						{Name: "Biotech Specialist", Description: "Лабораторные исследования, генная инженерия и биофармацевтика.", MatchPercentage: 95},
						{Name: "Medical Doctor", Description: "Клиническая практика и лечебная деятельность.", MatchPercentage: 83},
						{Name: "Data Scientist", Description: "Биоинформатика и анализ генетических кодов.", MatchPercentage: 75},
					},
					Grants: []Recommendation{
						{Name: "Грант Назарбаев Университета (NU)", Description: "Полные исследовательские гранты школы наук NU.", MatchPercentage: 94},
						{Name: "Yessenov Foundation Grant", Description: "Гранты на прохождение летних научных стажировок в лабораториях мира.", MatchPercentage: 90},
						{Name: "Болашак", Description: "Обучение биотехнологиям за рубежом.", MatchPercentage: 86},
					},
				}
			} else {
				result = AnalysisResult{
					TopProf: "Medical Doctor",
					Summary: "Твоя высокая эмпатия и научный интерес ведут тебя к самой благородной профессии. Быть врачом — это сложный путь, но он гарантирует колоссальную востребованность и реальное спасение жизней.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Подготовка к ЕНТ (Биология/Химия)", Description: "Пройди углубленные курсы по анатомии и общей химии для сдачи профильных тестов на высший балл."},
						{Step: "2", Duration: "2 месяца", Title: "Первая медицинская помощь и волонтерство", Description: "Пройди курсы первой помощи от Красного Полумесяца и поработай волонтером в медицинских пунктах."},
						{Step: "3", Duration: "6 месяцев", Title: "Конкурс на грант и поступление", Description: "Подай документы в медицинские вузы на государственный грант (Асфендиярова, МУА) или программы NU."},
					},
					Professions: []Recommendation{
						{Name: "Medical Doctor", Description: "Лечение пациентов, хирургия, терапия и педиатрия.", MatchPercentage: 96},
						{Name: "Biotech Specialist", Description: "Разработка инновационных лекарственных препаратов.", MatchPercentage: 81},
						{Name: "Civil Engineer", Description: "Проектирование безопасной экологичной инфраструктуры.", MatchPercentage: 65},
					},
					Grants: []Recommendation{
						{Name: "Грант Назарбаев Университета (NU)", Description: "Школа медицины Назарбаев Университета (NUSOM).", MatchPercentage: 95},
						{Name: "Болашак", Description: "Магистратура и резидентура за рубежом в ведущих клиниках мира.", MatchPercentage: 90},
						{Name: "Turkiye Burslari", Description: "Обучение медицине на гранте в Турции.", MatchPercentage: 81},
					},
				}
			}

		case "engineering":
			if leisure == "gadgets" {
				result = AnalysisResult{
					TopProf: "Robotics Engineer",
					Summary: "Роботы — это будущее промышленности. Твоя техническая страсть к гаджетам в сочетании с инженерным мышлением позволит тебе программировать контроллеры и собирать сложные автоматизированные системы.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Основы электроники и Arduino", Description: "Изучи основы схемотехники, закон Ома и начни собирать первые проекты на базе Arduino."},
						{Step: "2", Duration: "3 месяца", Title: "Программирование микроконтроллеров", Description: "Изучи базовый C/C++ для написания прошивок и пойми принципы работы датчиков и сервоприводов."},
						{Step: "3", Duration: "6 месяцев", Title: "3D-моделирование и CAD-системы", Description: "Научись моделировать детали в SolidWorks или Fusion 360, распечатай их на 3D-принтере и собери своего первого робота."},
					},
					Professions: []Recommendation{
						{Name: "Robotics Engineer", Description: "Проектирование и программирование автоматизированных роботизированных систем.", MatchPercentage: 94},
						{Name: "Civil Engineer", Description: "Расчет прочности конструкций и чертежи зданий.", MatchPercentage: 83},
						{Name: "Software Engineer", Description: "Написание программного обеспечения для управления железом.", MatchPercentage: 80},
					},
					Grants: []Recommendation{
						{Name: "KBTU Olympiad Grant", Description: "Гранты в КБТУ на мехатронику и робототехнику.", MatchPercentage: 91},
						{Name: "AITU Talent Grant", Description: "Гранты на инженерные IT-направления в Astana IT University.", MatchPercentage: 87},
						{Name: "Yessenov Foundation Grant", Description: "Гранты на стажировки в технических хабах мира.", MatchPercentage: 82},
					},
				}
			} else {
				result = AnalysisResult{
					TopProf: "Civil Engineer",
					Summary: "Классическая и фундаментальная инженерная специальность. Твой интерес к физике и конструированию позволит тебе проектировать современные города, жилые комплексы, дороги и мосты.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Основы физики и механики", Description: "Углубись в разделы статики и сопротивления материалов (сопромат) для понимания распределения нагрузок."},
						{Step: "2", Duration: "3 месяца", Title: "Программы черчения (AutoCAD/Revit)", Description: "Научись создавать двухмерные чертежи и строить 3D-информационные модели зданий (BIM)."},
						{Step: "3", Duration: "6 месяцев", Title: "Стажировка на стройплощадке", Description: "Пройди летнюю стажировку в строительной или архитектурной компании, чтобы увидеть работу чертежей вживую."},
					},
					Professions: []Recommendation{
						{Name: "Civil Engineer", Description: "Расчет нагрузок, составление смет и руководство строительными объектами.", MatchPercentage: 95},
						{Name: "Robotics Engineer", Description: "Проектирование манипуляторов и механизмов.", MatchPercentage: 80},
						{Name: "UX/UI Designer", Description: "Проектирование эргономики пространств (дизайн интерьеров).", MatchPercentage: 70},
					},
					Grants: []Recommendation{
						{Name: "Грант Назарбаев Университета (NU)", Description: "Обучение в Школе инженерии и цифровых наук NU.", MatchPercentage: 90},
						{Name: "Болашак", Description: "Магистратура в лучших строительных и архитектурных школах мира.", MatchPercentage: 85},
						{Name: "Turkiye Burslari", Description: "Полные гранты на гражданскую инженерию в Турции.", MatchPercentage: 81},
					},
				}
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

ПРАВИЛА И ИНТЕРВЬЮ ДЛЯ ROADMAP (Карьерного пути):
1. Если пользователь просит составить карьерный путь (или Roadmap), ты НЕ имеешь права генерировать его сразу.
2. Сначала задай пользователю 2-3 уточняющих вопроса по очереди. Спроси про его текущую базу, сколько времени он готов уделять учебе и какую конкретную цель он преследует (например, фриланс, стартап или работа в BigTech).
3. Как только пользователь ответит, сгенерируй ПОДРОБНЫЙ, ДЛИННЫЙ И КРУТОЙ роадмап из 5-8 шагов.
4. Каждый шаг в "roadmap" ОБЯЗАТЕЛЬНО должен быть детальным: пиши конкретные книги, технологии, фреймворки, методы и практические проекты, которые нужно реализовать на этом этапе.
5. Возвращай "roadmap" в JSON-ответе только тогда, когда пользователь ответил на твои вопросы.

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

ПРИМЕР ИДЕАЛЬНОГО ЗАПОЛНЕНИЯ CV (Подражай этому качеству):
{
  "name": "Алихан Сейткалиев",
  "contact": "+7 (777) 123-4567 | alikhan@example.com | github.com/alikhan | linkedin.com/in/alikhan",
  "tagline": "Junior Go Developer | MySQL | Docker",
  "summary": "Разработчик с опытом создания высоконагруженных бэкенд-систем на Go. Специализируюсь на проектировании REST API и работе с реляционными СУБД.",
  "experience": [
    {
      "company": "Платформа Talap AI",
      "role": "Lead Go Developer",
      "period": "Декабрь 2025 - Февраль 2026",
      "description": "Разработка ядра профориентационной платформы для подбора грантов и составления роадмапов.",
      "achievements": [
        "Спроектировал и реализовал REST API на базе Gin, сократив время отклика сервера до 80мс.",
        "Интегрировал OpenRouter API для чата, снизив стоимость генерации ответов на 30%%.",
        "Переписал слой БД с PostgreSQL на MySQL и настроил автосид данных для 30 грантов."
      ],
      "stack": "Go (Golang), GORM, MySQL, Gin Web Framework, OpenRouter API"
    }
  ],
  "education": [
    {
      "school": "Astana IT University",
      "degree": "Бакалавр Информационных Систем (GPA 3.8/4.0)",
      "period": "2023 - 2027"
    }
  ],
  "skills": ["Go (Golang)", "SQL", "MySQL", "Gin", "Git", "REST API", "Docker", "Algorithms"],
  "achievements_list": [
    "1-е место на хакатоне AITU Smart City Hackathon 2025 в категории 'Умная инфраструктура'",
    "Призер олимпиады KBTU Open 2024 по программированию (Топ-20 команд)"
  ],
  "languages": "Русский (Родной), Казахский (Свободный), Английский (IELTS 7.0)",
  "hobbies": "Робототехника, олимпиадное программирование, чтение книг по архитектуре ПО"
}
 
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
