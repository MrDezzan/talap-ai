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
		return "deepseek/deepseek-v4-pro-0813"
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
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	MatchPercentage int      `json:"match_percentage"`
	Subtitle        string   `json:"subtitle,omitempty"`
	DeadlineDays    int      `json:"deadline_days,omitempty"`
	Tone            string   `json:"tone,omitempty"`
	Country         string   `json:"country,omitempty"`
	Amount          string   `json:"amount,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

type CriteriaScore struct {
	Criterion string `json:"criterion"`
	Score     int    `json:"score"`
	Note      string `json:"note"`
}

type AnalysisResult struct {
	TopProf         string           `json:"top_profession"`
	TopProfCategory string           `json:"top_profession_category"`
	Summary         string           `json:"summary"`
	Roadmap         []RoadmapStep    `json:"roadmap"`
	Professions     []Recommendation `json:"professions"`
	Grants          []Recommendation `json:"grants"`
	IsFallback      bool             `json:"is_fallback"`
	FallbackNote    string           `json:"fallback_note,omitempty"`
	Breakdown       []CriteriaScore  `json:"breakdown"`
}

// categoryDisplayNames maps the internal scoring-category key to the
// real "category" value stored on models.Profession rows, so the LLM
// path and the fallback path both ground their output in real DB
// records instead of inventing names.
var categoryDisplayNames = map[string]string{
	"it":             "IT и данные",
	"design":         "Дизайн",
	"business":       "Бизнес",
	"medicine":       "Медицина",
	"engineering":    "Инженерия",
	"agriculture":    "Сельское хозяйство и ветеринария",
	"trades":         "Рабочие и технические специальности",
	"law":            "Право",
	"humanities":     "Гуманитарные науки и социальная работа",
	"education":      "Педагогика",
	"public_service": "Государственная служба и правопорядок",
	"sports":         "Спорт",
	"arts":           "Творческие профессии вне цифры",
	"transport":      "Транспорт и логистика",
	"construction":   "Строительство",
	"tourism":        "Туризм и гостиничное дело",
}

// categoryGrantKeywords maps the internal category key to a keyword that
// tends to appear in a real models.Grant's Tags for that field. models.Grant
// has no dedicated Category column, so this is a best-effort match against
// its serialized Tags text used to ground grant names without inventing them.
var categoryGrantKeywords = map[string]string{
	"it":             "IT",
	"design":         "Дизайн",
	"business":       "Бизнес",
	"medicine":       "Медицина",
	"engineering":    "Инженерия",
	"agriculture":    "Сельское хозяйство",
	"trades":         "Рабоч",
	"law":            "Юриспруденция",
	"humanities":     "Языки",
	"education":      "Педагогика",
	"public_service": "Госслужба",
	"sports":         "Спорт",
	"arts":           "Творчество",
	"transport":      "Транспорт",
	"construction":   "Строительство",
	"tourism":        "Туризм",
}

// fetchGroundingRecords pulls a handful of real professions and grants for
// the given category, so LLM prompts and fallback roadmap text can be
// grounded in real names instead of inventing them.
func fetchGroundingRecords(db *gorm.DB, category, realCategoryName string) ([]models.Profession, []models.Grant) {
	var professions []models.Profession
	var grants []models.Grant
	if db == nil {
		return professions, grants
	}
	if realCategoryName != "" {
		db.Where("category = ?", realCategoryName).Order("hot DESC, match_percentage DESC").Limit(5).Find(&professions)
	}
	if kw, ok := categoryGrantKeywords[category]; ok && kw != "" {
		db.Where("tags LIKE ?", "%"+kw+"%").Order("match_percentage DESC").Limit(5).Find(&grants)
	}
	if len(grants) == 0 {
		db.Order("match_percentage DESC").Limit(5).Find(&grants)
	}
	return professions, grants
}

func orDash(s string) string {
	if s == "" {
		return "не указано"
	}
	return s
}

// tryLLMAnalysis is the primary analysis path: it asks the LLM to produce a
// full AnalysisResult grounded in the user's quiz answers, the local
// rule-based scoring hint, and real profession/grant names pulled from the
// DB for the top-scoring category. Returns an error whenever the LLM is
// unavailable, times out, or returns something that doesn't parse into a
// usable AnalysisResult, so the caller can fall back to local rules.
func tryLLMAnalysis(db *gorm.DB, answers map[string]any, category, realCategoryName string, scores map[string]int) (*AnalysisResult, error) {
	client, err := openaiClient()
	if err != nil {
		return nil, err
	}

	professions, grants := fetchGroundingRecords(db, category, realCategoryName)

	var profLines strings.Builder
	for _, p := range professions {
		profLines.WriteString(fmt.Sprintf("- %s (категория: %s): %s\n", p.Name, p.Category, p.Description))
	}
	if profLines.Len() == 0 {
		profLines.WriteString("(нет точных совпадений в базе, придумай реалистичное, но не гарантированное название специальности)\n")
	}

	var grantLines strings.Builder
	for _, g := range grants {
		grantLines.WriteString(fmt.Sprintf("- %s (%s, %s): %s\n", g.Name, g.Country, g.Amount, g.Description))
	}
	if grantLines.Len() == 0 {
		grantLines.WriteString("(нет точных совпадений в базе)\n")
	}

	answersJSON, _ := json.Marshal(answers)
	scoresJSON, _ := json.Marshal(scores)

	prompt := fmt.Sprintf(`Ты — экспертный ИИ-профориентолог платформы Talap для школьников Казахстана.
Проведи честный и подробный анализ профиля ученика на основе его ответов на анкету из 12 вопросов.

ОТВЕТЫ УЧЕНИКА НА АНКЕТУ (JSON):
%s

ПОДСКАЗКА — локальный скоринг категорий по правилам (не единственный источник истины, используй его как ориентир, но опирайся в первую очередь на сами ответы):
%s
Категория-лидер по правилам: %s (%s)

РЕАЛЬНЫЕ ПРОФЕССИИ ИЗ БАЗЫ ДАННЫХ ДЛЯ ЭТОЙ КАТЕГОРИИ (используй ТОЛЬКО эти имена в top_profession и professions, не выдумывай другие названия, если они есть в списке):
%s

РЕАЛЬНЫЕ ГРАНТЫ ИЗ БАЗЫ ДАННЫХ (используй ТОЛЬКО эти имена в grants и в roadmap при упоминании грантов/вузов, если они подходят; если ни один не подходит, используй общую формулировку без выдуманных названий):
%s

ТРЕБОВАНИЯ:
1. Верни развернутый анализ, честно объясняющий, почему выбрана именно эта профессия/направление, ссылаясь на конкретные ответы ученика.
2. Roadmap — 5-7 конкретных шагов от подготовки до трудоустройства. Если упоминаешь вуз, колледж или грант — используй только реальные названия из списков выше.
3. В поле "breakdown" верни ровно 4 критерия, каждый с полями criterion, score (0-100) и note (одно предложение, почему такой балл, со ссылкой на конкретные ответы):
   - "Интересы и склонности"
   - "Формат работы (люди/данные/руки)"
   - "Отношение к риску и стабильности"
   - "Творческий vs аналитический склад"
4. Не выставляй "is_fallback" и "fallback_note" — они не нужны в этом ответе (оставь false/пусто).

ОТВЕЧАЙ СТРОГО В JSON СО СЛЕДУЮЩЕЙ СТРУКТУРОЙ (без markdown, без пояснений вне JSON):
{
  "top_profession": "...",
  "top_profession_category": "...",
  "summary": "...",
  "roadmap": [{"step": "1", "duration": "...", "title": "...", "description": "..."}],
  "professions": [{"name": "...", "description": "...", "match_percentage": 90}],
  "grants": [{"name": "...", "description": "...", "match_percentage": 90}],
  "breakdown": [{"criterion": "...", "score": 80, "note": "..."}]
}`, string(answersJSON), string(scoresJSON), category, realCategoryName, profLines.String(), grantLines.String())

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: llmModel(),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: prompt},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	})
	if err != nil {
		return nil, fmt.Errorf("llm request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("llm returned no choices")
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("invalid LLM JSON: %w", err)
	}
	if strings.TrimSpace(result.TopProf) == "" || len(result.Professions) == 0 {
		return nil, fmt.Errorf("incomplete LLM result")
	}

	result.IsFallback = false
	result.FallbackNote = ""
	return &result, nil
}

// computeFallbackBreakdown deterministically derives the 4 criteria scores
// from the same answer signals already used for rule-based scoring, so the
// fallback path stays explainable instead of showing a single opaque
// percentage.
func computeFallbackBreakdown(scores map[string]int, category string, subject, workType, environment, goal, problemSolving, leisure, projectRole, peopleInteraction, workspaceObjects, riskTolerance, natureScience, creativityType string) []CriteriaScore {
	total := 0
	for _, v := range scores {
		if v > 0 {
			total += v
		}
	}
	maxScore := scores[category]
	dominance := 50
	if total > 0 {
		dominance = int(float64(maxScore) / float64(total) * 100)
	}
	if dominance > 100 {
		dominance = 100
	}
	if dominance < 10 {
		dominance = 10
	}

	workFormatScore := 55
	if workType != "" {
		workFormatScore += 10
	}
	if peopleInteraction != "" {
		workFormatScore += 15
	}
	if workspaceObjects != "" {
		workFormatScore += 15
	}
	if workFormatScore > 100 {
		workFormatScore = 100
	}

	riskScore := 50
	if riskTolerance != "" {
		riskScore += 25
	}
	if goal != "" {
		riskScore += 15
	}
	if riskScore > 100 {
		riskScore = 100
	}

	creativeScore := 50
	if problemSolving == "intuition" || creativityType == "visual" || creativityType == "verbal" {
		creativeScore = 75
	}
	if problemSolving == "logic" || creativityType == "logic_art" {
		creativeScore = 30
	}
	if natureScience != "" || creativityType != "" {
		if creativeScore < 100 {
			creativeScore += 5
		}
	}

	return []CriteriaScore{
		{
			Criterion: "Интересы и склонности",
			Score:     dominance,
			Note:      fmt.Sprintf("Любимый предмет (%s) и увлечения (%s) сильнее всего указывают на направление «%s».", orDash(subject), orDash(leisure), category),
		},
		{
			Criterion: "Формат работы (люди/данные/руки)",
			Score:     workFormatScore,
			Note:      fmt.Sprintf("Ответы о формате работы (%s), общении с людьми (%s) и объектах труда (%s) показывают, тянет ли тебя к людям, данным или практической работе руками.", orDash(workType), orDash(peopleInteraction), orDash(workspaceObjects)),
		},
		{
			Criterion: "Отношение к риску и стабильности",
			Score:     riskScore,
			Note:      fmt.Sprintf("Ответы про отношение к риску (%s) и главную цель (%s) говорят о том, что тебе ближе — стабильность или вызовы.", orDash(riskTolerance), orDash(goal)),
		},
		{
			Criterion: "Творческий vs аналитический склад",
			Score:     creativeScore,
			Note:      fmt.Sprintf("Способ решения задач (%s) и тип творчества, который тебе ближе (%s), показывают баланс между логикой и креативностью.", orDash(problemSolving), orDash(creativityType)),
		},
	}
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
		peopleInteraction := ""
		workspaceObjects := ""
		riskTolerance := ""
		natureScience := ""
		creativityType := ""

		if answers != nil {
			if s, ok := answers["subject"].(string); ok { subject = s }
			if w, ok := answers["work_type"].(string); ok { workType = w }
			if e, ok := answers["environment"].(string); ok { environment = e }
			if g, ok := answers["goal"].(string); ok { goal = g }
			if p, ok := answers["problem_solving"].(string); ok { problemSolving = p }
			if l, ok := answers["leisure"].(string); ok { leisure = l }
			if r, ok := answers["project_role"].(string); ok { projectRole = r }
			if pi, ok := answers["people_interaction"].(string); ok { peopleInteraction = pi }
			if wo, ok := answers["workspace_objects"].(string); ok { workspaceObjects = wo }
			if rt, ok := answers["risk_tolerance"].(string); ok { riskTolerance = rt }
			if ns, ok := answers["nature_science"].(string); ok { natureScience = ns }
			if ct, ok := answers["creativity_type"].(string); ok { creativityType = ct }
		}

		// Scoring categories
		itScore := 0
		designScore := 0
		businessScore := 0
		medicineScore := 0
		engineeringScore := 0
		agricultureScore := 0
		tradesScore := 0
		lawScore := 0
		humanitiesScore := 0
		educationScore := 0
		publicServiceScore := 0
		sportsScore := 0
		artsScore := 0
		transportScore := 0
		constructionScore := 0
		tourismScore := 0

		// 1. Subject scoring
		switch subject {
		case "math":
			itScore += 20
			engineeringScore += 15
			businessScore += 5
			constructionScore += 10
		case "art":
			designScore += 20
			artsScore += 15
		case "human":
			businessScore += 20
			designScore += 5
			humanitiesScore += 20
			educationScore += 15
			lawScore += 15
			publicServiceScore += 10
			tourismScore += 10
		case "science":
			medicineScore += 20
			engineeringScore += 15
			agricultureScore += 15
		}

		// 2. Work type scoring
		switch workType {
		case "team":
			businessScore += 10
			medicineScore += 10
			educationScore += 10
			publicServiceScore += 10
			sportsScore += 10
		case "solo":
			itScore += 10
			engineeringScore += 10
			tradesScore += 10
			artsScore += 5
		case "lead":
			businessScore += 15
			publicServiceScore += 10
		case "create":
			itScore += 10
			designScore += 15
			engineeringScore += 10
			artsScore += 15
			constructionScore += 5
		}

		// 3. Environment scoring
		switch environment {
		case "office":
			businessScore += 10
			itScore += 5
			lawScore += 10
			publicServiceScore += 5
		case "remote":
			itScore += 15
			designScore += 15
			artsScore += 5
		case "travel":
			businessScore += 15
			tourismScore += 20
			transportScore += 15
		case "lab":
			medicineScore += 15
			engineeringScore += 15
			agricultureScore += 15
		}

		// 4. Goal scoring
		switch goal {
		case "money":
			itScore += 10
			businessScore += 15
			lawScore += 10
			transportScore += 5
		case "impact":
			medicineScore += 15
			engineeringScore += 5
			educationScore += 20
			publicServiceScore += 20
			humanitiesScore += 15
			agricultureScore += 5
		case "fame":
			designScore += 15
			businessScore += 5
			artsScore += 15
			sportsScore += 10
		case "growth":
			itScore += 10
			engineeringScore += 10
			medicineScore += 5
			tradesScore += 10
			constructionScore += 5
		}

		// 5. Problem solving scoring
		switch problemSolving {
		case "logic":
			itScore += 10
			engineeringScore += 10
			medicineScore += 5
			lawScore += 15
		case "intuition":
			designScore += 15
			artsScore += 10
		case "people":
			businessScore += 10
			medicineScore += 10
			humanitiesScore += 15
			educationScore += 10
			publicServiceScore += 10
			tourismScore += 10
		case "practical":
			itScore += 5
			engineeringScore += 10
			designScore += 5
			tradesScore += 20
			constructionScore += 15
			transportScore += 15
			agricultureScore += 10
		}

		// 6. Leisure scoring
		switch leisure {
		case "gadgets":
			itScore += 15
			engineeringScore += 15
			tradesScore += 10
			transportScore += 5
		case "draw":
			designScore += 20
			artsScore += 20
		case "books":
			medicineScore += 15
			engineeringScore += 10
			lawScore += 10
			humanitiesScore += 15
		case "games":
			businessScore += 15
			designScore += 5
			sportsScore += 10
		}

		// 7. Project role scoring
		switch projectRole {
		case "builder":
			itScore += 15
			engineeringScore += 15
			designScore += 5
			tradesScore += 15
			constructionScore += 15
		case "presenter":
			businessScore += 15
			designScore += 10
			educationScore += 10
			tourismScore += 10
		case "manager":
			businessScore += 20
			publicServiceScore += 15
			sportsScore += 10
		case "researcher":
			medicineScore += 20
			engineeringScore += 10
			agricultureScore += 15
			lawScore += 5
		}

		// 8. Workspace objects scoring
		switch workspaceObjects {
		case "code_data":
			itScore += 15
		case "materials":
			constructionScore += 10
			tradesScore += 10
			engineeringScore += 10
		case "texts_law":
			lawScore += 15
			humanitiesScore += 10
		case "media_art":
			artsScore += 15
			designScore += 10
		}

		// 9. Risk tolerance scoring
		switch riskTolerance {
		case "stable":
			publicServiceScore += 15
			educationScore += 10
		case "flexible":
			designScore += 10
			businessScore += 10
		case "adventure":
			businessScore += 10
			sportsScore += 15
		case "scientific":
			medicineScore += 15
			engineeringScore += 10
		}

		// 10. Nature/science scoring
		switch natureScience {
		case "bio_tech":
			medicineScore += 15
			agricultureScore += 10
		case "geo_eco":
			agricultureScore += 15
			tourismScore += 10
		case "tech_only":
			itScore += 10
			engineeringScore += 10
		case "urban":
			constructionScore += 15
		}

		// 11. Creativity type scoring
		switch creativityType {
		case "logic_art":
			itScore += 10
			engineeringScore += 10
		case "visual":
			designScore += 15
			artsScore += 10
		case "verbal":
			humanitiesScore += 15
			educationScore += 10
		case "business_ideas":
			businessScore += 15
		}

		// 12. People interaction scoring
		switch peopleInteraction {
		case "help":
			medicineScore += 15
			educationScore += 10
		case "teach":
			educationScore += 15
		case "negotiate":
			businessScore += 10
			lawScore += 10
		case "none":
			itScore += 10
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
		if agricultureScore > maxScore {
			maxScore = agricultureScore
			category = "agriculture"
		}
		if tradesScore > maxScore {
			maxScore = tradesScore
			category = "trades"
		}
		if lawScore > maxScore {
			maxScore = lawScore
			category = "law"
		}
		if humanitiesScore > maxScore {
			maxScore = humanitiesScore
			category = "humanities"
		}
		if educationScore > maxScore {
			maxScore = educationScore
			category = "education"
		}
		if publicServiceScore > maxScore {
			maxScore = publicServiceScore
			category = "public_service"
		}
		if sportsScore > maxScore {
			maxScore = sportsScore
			category = "sports"
		}
		if artsScore > maxScore {
			maxScore = artsScore
			category = "arts"
		}
		if transportScore > maxScore {
			maxScore = transportScore
			category = "transport"
		}
		if constructionScore > maxScore {
			maxScore = constructionScore
			category = "construction"
		}
		if tourismScore > maxScore {
			maxScore = tourismScore
			category = "tourism"
		}

		scores := map[string]int{
			"it":             itScore,
			"design":         designScore,
			"business":       businessScore,
			"medicine":       medicineScore,
			"engineering":    engineeringScore,
			"agriculture":    agricultureScore,
			"trades":         tradesScore,
			"law":            lawScore,
			"humanities":     humanitiesScore,
			"education":      educationScore,
			"public_service": publicServiceScore,
			"sports":         sportsScore,
			"arts":           artsScore,
			"transport":      transportScore,
			"construction":   constructionScore,
			"tourism":        tourismScore,
		}
		realCategoryName := categoryDisplayNames[category]

		// Primary path: try the LLM first, grounded in the rule-based scoring
		// hint and real profession/grant names for the leading category.
		var result AnalysisResult
		llmResult, llmErr := tryLLMAnalysis(db, answers, category, realCategoryName, scores)
		if llmErr == nil && llmResult != nil {
			result = *llmResult
		} else {
			if llmErr != nil {
				log.Printf("AnalyzeProfile: LLM analysis unavailable, falling back to rules: %v", llmErr)
			}
			result = fallbackAnalysisResult(db, category, leisure, projectRole, environment, problemSolving)
			result.IsFallback = true
			result.FallbackNote = "Полный AI-анализ временно недоступен — показан упрощённый результат на основе твоих ответов. Попробуй пройти тест ещё раз позже."
			result.Breakdown = computeFallbackBreakdown(scores, category, subject, workType, environment, goal, problemSolving, leisure, projectRole, peopleInteraction, workspaceObjects, riskTolerance, natureScience, creativityType)
		}

		db.FirstOrCreate(&portfolio, models.Portfolio{UserID: uid})
		enrichGrants(db, result.Grants)
		resJSON, _ := json.Marshal(result)
		db.Model(&portfolio).Updates(models.Portfolio{
			Interests:   portfolio.Interests,
			QuizAnswers: portfolio.QuizAnswers,
			AIResult:    resJSON,
			UpdatedAt:   time.Now(),
		})

		c.JSON(http.StatusOK, result)
	}
}

func fallbackAnalysisResult(db *gorm.DB, category, leisure, projectRole, environment, problemSolving string) AnalysisResult {
	var result AnalysisResult
	switch category {
		case "it":
			if leisure == "gadgets" && projectRole == "builder" {
				result = AnalysisResult{
					TopProf: "AI/ML Specialist",
					TopProfCategory: "IT и данные",
					Summary: "Отличный выбор! Твоя страсть к инновациям и желание создавать интеллектуальные алгоритмы идеально подходят для сферы ИИ. Профессия разработчика машинного обучения имеет невероятную скорость роста спроса.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1-2 месяца", Title: "Python, ООП и структуры данных", Description: "Изучи Python на продвинутом уровне, работу с классами, ООП и структурами данных (стек, очередь, деревья)."},
						{Step: "2", Duration: "2 месяца", Title: "Математика для Data Science", Description: "Освежи знания линейной алгебры (матрицы), теории вероятностей и математического анализа (градиентный спуск)."},
						{Step: "3", Duration: "3 месяца", Title: "Классическое машинное обучение", Description: "Научись работать с Jupyter Notebook, библиотеками Pandas, NumPy, Matplotlib и алгоритмами Scikit-Learn (регрессии, деревья решений)."},
						{Step: "4", Duration: "3 месяца", Title: "Глубокое обучение и Нейросети", Description: "Переходи к фреймворкам PyTorch или TensorFlow. Реализуй проекты по компьютерному зрению (CNN) или NLP."},
						{Step: "5", Duration: "2 месяца", Title: "NLP, LLM и деплой моделей", Description: "Освой HuggingFace, основы дообучения (fine-tuning) LLM и развертывание моделей через FastAPI и Docker."},
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
					TopProfCategory: "IT и данные",
					Summary: "Твой аналитический склад ума в сочетании со стремлением работать самостоятельно отлично подходит для разработки ПО. Разработка бэкенда дает полную свободу в выборе места работы.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Базовые алгоритмы и синтаксис", Description: "Освой основы программирования на Go или Python, разберись с типами данных, циклами и простыми алгоритмами."},
						{Step: "2", Duration: "2 месяца", Title: "Базы данных и SQL", Description: "Изучи проектирование реляционных баз данных, работу с PostgreSQL, написание сложных запросов и индексы."},
						{Step: "3", Duration: "2 месяца", Title: "Разработка API и фреймворки", Description: "Изучи разработку REST API на современных фреймворках (Gin для Go или FastAPI для Python), авторизацию JWT и сессии."},
						{Step: "4", Duration: "2 месяца", Title: "Системный дизайн и Контейнеризация", Description: "Освой Docker, принципы построения микросервисной архитектуры и базовые очереди сообщений (Redis, RabbitMQ)."},
						{Step: "5", Duration: "2 месяца", Title: "GitHub, CI/CD и Пет-проекты", Description: "Создай полноценный проект (например, сервис для онлайн-кинотеатра), настрой CI/CD и выложи в портфолио."},
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
					TopProfCategory: "IT и данные",
					Summary: "Логика и внимание к деталям — главные качества белого хакера. Ты научишься мыслить как злоумышленник, чтобы выстраивать надежную защиту корпоративных систем от угроз.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Основы сетей и Linux", Description: "Освой администрирование Linux в терминале, пойми работу сетей (DNS, DHCP, стек протоколов TCP/IP)."},
						{Step: "2", Duration: "2 месяца", Title: "Криптография и Веб-безопасность", Description: "Изучи методы шифрования, симметричные/асимметричные ключи, уязвимости OWASP Top 10."},
						{Step: "3", Duration: "2 месяца", Title: "Инструменты анализа безопасности", Description: "Освой инструменты сканирования сетей и перехвата трафика (Nmap, Wireshark, Burp Suite, Metasploit)."},
						{Step: "4", Duration: "3 месяца", Title: "Практика на CTF и HackTheBox", Description: "Решай задания по безопасности, учись эксплуатировать уязвимости на платформах HackTheBox и TryHackMe."},
						{Step: "5", Duration: "3 месяца", Title: "Аудит и Подготовка к сертификатам", Description: "Изучи процессы аудита безопасности, основы DevSecOps и готовься к международному сертификату CompTIA Security+."},
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
					TopProfCategory: "IT и данные",
					Summary: "Каждый использует мобильные приложения ежедневно. Твоя техническая подготовка позволит тебе создавать быстрые, удобные и красивые приложения для iOS или Android.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Язык программирования (Swift/Kotlin)", Description: "Изучи синтаксис Swift для iOS-разработки или Kotlin для Android-разработки, основы ООП."},
						{Step: "2", Duration: "2 месяца", Title: "Верстка экранов и дизайн-системы", Description: "Научись создавать современные интерфейсы с использованием фреймворков SwiftUI (iOS) или Jetpack Compose (Android)."},
						{Step: "3", Duration: "2 месяца", Title: "Работа с сетью и Базами данных", Description: "Изучи отправку HTTP-запросов, парсинг JSON-ответов REST API, локальные базы данных Room (Android) или SwiftData (iOS)."},
						{Step: "4", Duration: "2 месяца", Title: "Архитектура и Внедрение зависимостей", Description: "Освой паттерны MVVM и Clean Architecture, реактивное программирование и внедрение зависимостей (Dagger/Hilt/Swinject)."},
						{Step: "5", Duration: "3 месяца", Title: "Публикация первого приложения", Description: "Разработай полноценный пет-проект с авторизацией и сетью, опубликуй его в App Store или Google Play."},
					},
					Professions: []Recommendation{
						{Name: "Mobile Developer", Description: "Создание приложений для смартфонов и таблиц.", MatchPercentage: 94},
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
					TopProfCategory: "Дизайн",
					Summary: "Идеальный баланс аналитики и творчества! Твоя способность мыслить логически поможет тебе создавать удобную структуру интерфейсов, а эстетический вкус — делать их привлекательными.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Основы Figma и Инструменты", Description: "Освой основы создания компонентов, автолейаутов, вариантов, стилей и адаптивной верстки в Figma."},
						{Step: "2", Duration: "2 месяца", Title: "UX-исследования и CJM", Description: "Научись проводить коридорные тестирования, интервью, строить карты CJM и проектировать логику интерфейсов."},
						{Step: "3", Duration: "2 месяца", Title: "Визуальный дизайн и Типографика", Description: "Изучи теорию цвета, правила контраста, модульные сетки и работу с типографикой для веба и мобильных платформ."},
						{Step: "4", Duration: "2 месяца", Title: "Проектирование Web и Mobile", Description: "Спроектируй интерфейсы для сложной веб-платформы и адаптивного мобильного приложения, следуя гайдлайнам (Human Interface / Material Design)."},
						{Step: "5", Duration: "2 месяца", Title: "Портфолио на Behance и Кейсы", Description: "Переработай дизайн неудобного государственного/бизнес-сервиса, оформи кейс с описанием процесса и опубликуй на Behance и Dribbble."},
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
					TopProfCategory: "Дизайн",
					Summary: "Твоя креативность и понимание людей делают тебя отличным контент-стратегом. Ты сможешь легко находить общий язык с аудиторией и создавать вирусный визуальный и текстовый контент.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Сторителлинг и Мобильная съемка", Description: "Пойми принципы удержания внимания, раскадровки, монтажа динамичных видео и работы с естественным светом."},
						{Step: "2", Duration: "2 месяца", Title: "Монтаж и Аудиоэффекты", Description: "Освой мобильный и десктопный монтаж (CapCut, Premiere Pro), цветокоррекцию, саунд-дизайн и динамические субтитры."},
						{Step: "3", Duration: "2 месяца", Title: "Алгоритмы платформ и Аналитика", Description: "Изучи алгоритмы рекомендаций в TikTok, Instagram Reels и YouTube Shorts. Научись анализировать CTR и удержание аудитории."},
						{Step: "4", Duration: "2 месяца", Title: "Копирайтинг и Прогревы", Description: "Освой основы маркетингового копирайтинга, разработку контент-планов и написание сценариев для прогревов и интеграций."},
						{Step: "5", Duration: "2 месяца", Title: "Личный бренд и Таргетинг", Description: "Запусти рекламу на свои тестовые проекты, собери портфолио с кейсами роста охватов и начни работу с первыми коммерческими брендами."},
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
					TopProfCategory: "Бизнес",
					Summary: "Отличные организаторские качества и умение сплотить команду делают тебя идеальным руководителем продуктов. Ты будешь направлять команду разработки к общей цели.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Методологии Agile & Scrum", Description: "Пойми, как организуются процессы разработки в IT-компаниях по гибким методологиям (Scrum, Kanban, Jira)."},
						{Step: "2", Duration: "2 месяца", Title: "CustDev и Проверка гипотез", Description: "Изучи основы Customer Development, проведение глубинных интервью с пользователями и проектирование MVP."},
						{Step: "3", Duration: "2 месяца", Title: "Продуктовая аналитика", Description: "Разберись в воронках (AARRR), расчете юнит-экономики (LTV, CAC), когортном анализе и метриках удержания (Retention)."},
						{Step: "4", Duration: "2 месяца", Title: "Приоритизация бэклога", Description: "Изучи фреймворки приоритизации задач (RICE, ICE, WSJF) и построение дорожных карт продукта (Gantt, ProductPlan)."},
						{Step: "5", Duration: "2 месяца", Title: "Запуск No-code проекта", Description: "Создай и протестируй прототип приложения на Glide или Bubble, собери первые метрики и оформи в портфолио."},
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
					TopProfCategory: "Бизнес",
					Summary: "Сочетание прагматичности, любви к математике и понимания бизнес-процессов — залог успеха в сфере финансов. Ты будешь анализировать инвестиции, риски и выстраивать стратегии доходности.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Финансовый Excel", Description: "Изучи формулы, ВПР, индексы, сводные таблицы и диаграммы Excel на экспертном уровне."},
						{Step: "2", Duration: "2 месяца", Title: "Корпоративная отчетность", Description: "Разберись в трех основных отчетах: P&L (прибыли/убытки), Cash Flow (движение средств) и Balance Sheet (баланс)."},
						{Step: "3", Duration: "2 месяца", Title: "Финансовый анализ компании", Description: "Изучи коэффициенты ликвидности, рентабельности и финансовой устойчивости компании."},
						{Step: "4", Duration: "3 месяца", Title: "Финансовое моделирование", Description: "Освой оценку инвестиционных проектов, дисконтирование денежных потоков (DCF, NPV, IRR) и анализ чувствительности рисков."},
						{Step: "5", Duration: "3 месяца", Title: "Сертификация CFA и Стажировки", Description: "Начни подготовку к базовому уровню международной сертификации CFA и пройди отбор на стажировку в Big 4 или консалтинг."},
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
					TopProfCategory: "Бизнес",
					Summary: "Стратегический взгляд и глубокое понимание человеческой психологии позволят тебе вести крупные рекламные проекты. Ты будешь определять, как именно аудитория видит продукты и услуги.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "1 месяц", Title: "Теория маркетинга и Позиционирование", Description: "Изучи концепцию 4P, позиционирование бренда, анализ конкурентов и сегментацию целевой аудитории."},
						{Step: "2", Duration: "3 месяца", Title: "Digital-маркетинг и Таргетинг", Description: "Освой настройку контекстной рекламы (Google Ads), таргет в соцсетях, SEO-оптимизацию и веб-аналитику (Google Analytics)."},
						{Step: "3", Duration: "2 месяца", Title: "Воронки продаж и CRM", Description: "Научись строить сквозные воронки продаж, автоматизировать email-маркетинг и работать с CRM-системами."},
						{Step: "4", Duration: "2 месяца", Title: "Бренд-маркетинг и PR", Description: "Изучи создание медиа-стратегий, работу с инфлюенсерами, управление репутацией бренда и креативный копирайтинг."},
						{Step: "5", Duration: "2 месяца", Title: "Экономика маркетинга и Сквозная аналитика", Description: "Научись считать окупаемость инвестиций (ROMI), управлять маркетинговыми бюджетами и оптимизировать стоимость лида."},
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
					TopProfCategory: "Медицина",
					Summary: "Медицина будущего! Твой исследовательский интерес к науке позволит тебе создавать инновации в лабораториях: новые вакцины, методы лечения рака и биотехнологические продукты.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Общая биология, химия и генетика", Description: "Изучи основы общей биологии, органической химии, принципы наследственности и основы генетики."},
						{Step: "2", Duration: "3 месяца", Title: "Молекулярная и клеточная биология", Description: "Изучи строение клетки, процессы репликации ДНК, транскрипции и трансляции, теорию ПЦР и секвенирования."},
						{Step: "3", Duration: "3 месяца", Title: "Биоинформатика и Анализ данных", Description: "Освой основы Python для обработки биологических последовательностей, работу с базами данных (NCBI, BLAST)."},
						{Step: "4", Duration: "3 месяца", Title: "Лабораторная практика и Правила GLP", Description: "Изучи стандарты лабораторной работы GLP, методы культивирования клеток и очистки белков (хроматография)."},
						{Step: "5", Duration: "3 месяца", Title: "Научный проект и Публикация", Description: "Под руководством научного руководителя напиши первую исследовательскую работу, подготовь постерный доклад."},
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
					TopProfCategory: "Медицина",
					Summary: "Твоя высокая эмпатия и научный интерес ведут тебя к самой благородной профессии. Быть врачом — это сложный путь, но он гарантирует колоссальную востребованность и реальное спасение жизней.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Подготовка к ЕНТ (Биология/Химия)", Description: "Пройди углубленные курсы по анатомии человека и общей химии для сдачи профильных тестов на высший балл."},
						{Step: "2", Duration: "2 месяца", Title: "Первая медицинская помощь и Волонтерство", Description: "Пройди курсы первой помощи от Красного Полумесяца и поработай волонтером в клиниках или домах престарелых."},
						{Step: "3", Duration: "2 года", Title: "Медицинский вуз: Базовые дисциплины", Description: "Поступи в медвуз на грант, освой анатомию, гистологию, биохимию и физиологию на первом-втором курсах."},
						{Step: "4", Duration: "2 года", Title: "Клинические дисциплины", Description: "Пройди пропедевтику внутренних болезней, хирургию, терапию и педиатрию в клиниках при университете."},
						{Step: "5", Duration: "2-3 года", Title: "Интернатура и Резидентура", Description: "Закончи бакалавриат, поступи в интернатуру и выбери узкую специализацию в резидентуре (хирургия, кардиология и др.)."},
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
					TopProfCategory: "Инженерия",
					Summary: "Роботы — это будущее промышленности. Твоя техническая страсть к гаджетам в сочетании с инженерным мышлением позволит тебе программировать контроллеры и собирать сложные автоматизированные системы.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2 месяца", Title: "Электротехника и Схемотехника", Description: "Изучи основы электричества, закон Ома, законы Кирхгофа и начни собирать первые схемы на макетной плате."},
						{Step: "2", Duration: "3 месяца", Title: "Программирование C/C++ (Arduino)", Description: "Изучи программирование на C/C++ для написания прошивок к микроконтроллерам AVR, STM32 или ESP32."},
						{Step: "3", Duration: "2 месяца", Title: "3D-моделирование и CAD-системы", Description: "Научись моделировать детали в SolidWorks или Fusion 360, пойми принципы 3D-печати и сборки механизмов."},
						{Step: "4", Duration: "3 месяца", Title: "Датчики, Сервоприводы и Управление", Description: "Разберись с теорией автоматического управления (ПИД-регуляторы), работой гироскопов, дальномеров и моторов."},
						{Step: "5", Duration: "3 месяца", Title: "Сборка Мобильного робота", Description: "Спроектируй, собери и запрограммируй своего колесного робота с обходом препятствий и управлением по Bluetooth."},
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
					TopProfCategory: "Инженерия",
					Summary: "Классическая и фундаментальная инженерная специальность. Твой интерес к физике и конструированию позволит тебе проектировать современные города, жилые комплексы, дороги и мосты.",
					Roadmap: []RoadmapStep{
						{Step: "1", Duration: "2-3 месяца", Title: "Механика грунтов и Сопромат", Description: "Изучи разделы теоретической механики, основы распределения нагрузок и сопротивления материалов (сопромат)."},
						{Step: "2", Duration: "2 месяца", Title: "Программы черчения (AutoCAD)", Description: "Научись создавать двухмерные чертежи, схемы и планы в AutoCAD, следуя строительным чертежным стандартам."},
						{Step: "3", Duration: "3 месяца", Title: "3D BIM-моделирование (Revit)", Description: "Освой основы технологии информационного моделирования зданий (BIM) в программе Autodesk Revit."},
						{Step: "4", Duration: "3 месяца", Title: "Строительные нормы (СНиП)", Description: "Изучи СНиПы и ГОСТы, методы расчета несущей способности фундаментов, перекрытий и стен под нагрузкой."},
						{Step: "5", Duration: "3 месяца", Title: "Стажировка на Строительной площадке", Description: "Пройди практику в проектной или строительной компании, чтобы ознакомиться с процессами надзора и ведения смет."},
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

		default:
			realCategoryName, ok := map[string]string{
				"agriculture":    "Сельское хозяйство и ветеринария",
				"trades":         "Рабочие и технические специальности",
				"law":            "Право",
				"humanities":     "Гуманитарные науки и социальная работа",
				"education":      "Педагогика",
				"public_service": "Государственная служба и правопорядок",
				"sports":         "Спорт",
				"arts":           "Творческие профессии вне цифры",
				"transport":      "Транспорт и логистика",
				"construction":   "Строительство",
				"tourism":        "Туризм и гостиничное дело",
			}[category]

			if !ok || db == nil {
				result = AnalysisResult{
					TopProf:         "Специалист",
					TopProfCategory: "",
					Summary:         "Мы пока не смогли подобрать точную рекомендацию для твоего профиля. Попробуй пройти тест еще раз или изучи каталог профессий самостоятельно.",
					Roadmap:         []RoadmapStep{},
					Professions:     []Recommendation{},
					Grants:          []Recommendation{},
				}
				break
			}

			var catProfessions []models.Profession
			db.Where("category = ?", realCategoryName).Order("hot DESC, match_percentage DESC").Limit(3).Find(&catProfessions)

			if len(catProfessions) == 0 {
				result = AnalysisResult{
					TopProf:         "Специалист",
					TopProfCategory: realCategoryName,
					Summary:         fmt.Sprintf("По твоим ответам ближе всего подходит направление «%s». Загляни в каталог профессий, чтобы изучить варианты в этой сфере подробнее.", realCategoryName),
					Roadmap:         []RoadmapStep{},
					Professions:     []Recommendation{},
					Grants:          []Recommendation{},
				}
				break
			}

			top := catProfessions[0]

			// Feature 3: ground the "how to apply" step in a real grant name
			// pulled from the DB for this category, when one exists.
			applyStepDescription := "Определи, нужен колледж, вуз или профессиональные курсы для входа в профессию, и запишись на подготовительные занятия по нужным предметам (ЕНТ/ОРТ при поступлении)."
			if kw, ok := categoryGrantKeywords[category]; ok && kw != "" {
				var relatedGrants []models.Grant
				db.Where("tags LIKE ?", "%"+kw+"%").Order("match_percentage DESC").Limit(1).Find(&relatedGrants)
				if len(relatedGrants) > 0 {
					applyStepDescription = fmt.Sprintf("Определи, нужен колледж, вуз или профессиональные курсы для входа в профессию. Обрати внимание на грант «%s» (%s) — он подходит для этого направления.", relatedGrants[0].Name, relatedGrants[0].Country)
				}
			}

			result = AnalysisResult{
				TopProf:         top.Name,
				TopProfCategory: realCategoryName,
				Summary:         fmt.Sprintf("По твоим ответам ближе всего подходит направление «%s». %s", top.Name, top.Description),
				Roadmap: []RoadmapStep{
					{Step: "1", Duration: "1-2 месяца", Title: "Знакомство со сферой", Description: fmt.Sprintf("Изучи, что входит в работу «%s» на практике: посмотри интервью с практикующими специалистами, дни открытых дверей в колледжах/вузах по этому направлению.", top.Name)},
					{Step: "2", Duration: "2-3 месяца", Title: "Базовые знания и навыки", Description: applyStepDescription},
					{Step: "3", Duration: "3-4 месяца", Title: "Практика и стажировка", Description: "Найди возможность попробовать себя на практике — стажировку, волонтёрство или подработку в смежной сфере, чтобы проверить интерес к профессии вживую."},
					{Step: "4", Duration: "2-3 месяца", Title: "Поступление или трудоустройство", Description: "Подготовь документы и подай заявку в выбранное учебное заведение или на подходящую вакансию/ученичество."},
					{Step: "5", Duration: "постоянно", Title: "Развитие в профессии", Description: "Продолжай развивать навыки через курсы повышения квалификации и профессиональное сообщество в этой сфере."},
				},
				Professions: []Recommendation{},
				Grants:      []Recommendation{},
			}
			for _, p := range catProfessions {
				result.Professions = append(result.Professions, Recommendation{Name: p.Name, Description: p.Description, MatchPercentage: p.MatchPercentage})
			}
		}
	return result
}

func enrichGrants(db *gorm.DB, grants []Recommendation) {
	for i, g := range grants {
		// First try database
		var dbGrant models.Grant
		if err := db.Where("LOWER(name) = LOWER(?)", g.Name).First(&dbGrant).Error; err == nil {
			grants[i].Subtitle = dbGrant.Subtitle
			grants[i].DeadlineDays = dbGrant.DeadlineDays
			grants[i].Tone = dbGrant.Tone
			grants[i].Country = dbGrant.Country
			grants[i].Amount = dbGrant.Amount
			continue
		}

		// Fallback to static rules
		switch g.Name {
		case "AITU Talent Grant":
			grants[i].Subtitle = "Астана, Astana IT University"
			grants[i].DeadlineDays = 25
			grants[i].Tone = "blue"
			grants[i].Country = "Казахстан"
			grants[i].Amount = "100%% грант"
		case "Google STEP Internship":
			grants[i].Subtitle = "Стажировка в Google"
			grants[i].DeadlineDays = 45
			grants[i].Tone = "blue"
			grants[i].Country = "США/Европа"
			grants[i].Amount = "$5,000/мес"
		case "KBTU Olympiad Grant":
			grants[i].Subtitle = "Алматы, КБТУ"
			grants[i].DeadlineDays = 14
			grants[i].Tone = "warn"
			grants[i].Country = "Казахстан"
			grants[i].Amount = "100%% грант"
		case "Грант Назарбаев Университета (NU)":
			grants[i].Subtitle = "Астана, NU"
			grants[i].DeadlineDays = 30
			grants[i].Tone = "blue"
			grants[i].Country = "Казахстан"
			grants[i].Amount = "100%% грант"
		case "Болашак":
			grants[i].Subtitle = "Обучение за рубежом"
			grants[i].DeadlineDays = 60
			grants[i].Tone = "blue"
			grants[i].Country = "Весь мир"
			grants[i].Amount = "100%% оплата"
		case "Turkiye Burslari":
			grants[i].Subtitle = "Обучение в Турции"
			grants[i].DeadlineDays = 12
			grants[i].Tone = "warn"
			grants[i].Country = "Турция"
			grants[i].Amount = "100%% грант + стипендия"
		case "SDU Internal Grant":
			grants[i].Subtitle = "Каскелен, СДУ"
			grants[i].DeadlineDays = 8
			grants[i].Tone = "warn"
			grants[i].Country = "Казахстан"
			grants[i].Amount = "до 100%% скидка"
		case "Chevening Scholarship":
			grants[i].Subtitle = "Магистратура в Великобритании"
			grants[i].DeadlineDays = 90
			grants[i].Tone = "blue"
			grants[i].Country = "Великобритания"
			grants[i].Amount = "100%% покрытие"
		case "DAAD Scholarship":
			grants[i].Subtitle = "Обучение в Германии"
			grants[i].DeadlineDays = 75
			grants[i].Tone = "blue"
			grants[i].Country = "Германия"
			grants[i].Amount = "Бесплатное обучение + €934/мес"
		default:
			// Fallbacks
			grants[i].Subtitle = "Государственный грант"
			grants[i].DeadlineDays = 30
			grants[i].Tone = "blue"
			grants[i].Country = "Казахстан"
			grants[i].Amount = "100%% грант"
		}
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

		// Token-drain protection: input.Message and input.History are fully
		// client-controlled and get forwarded almost verbatim into the LLM
		// request. Without limits here, a single authenticated request could
		// carry an arbitrarily large message or thousands of history entries
		// and burn through the OpenRouter/DeepSeek quota in one call.
		const (
			maxMessageChars      = 4000
			maxHistoryEntries    = 20
			maxHistoryEntryChars = 2000
		)
		if len([]rune(input.Message)) > maxMessageChars {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Сообщение слишком длинное (максимум %d символов)", maxMessageChars)})
			return
		}
		if len(input.History) > maxHistoryEntries {
			// Keep only the most recent entries — older context is the least
			// useful and the most likely to be padding in an abuse attempt.
			input.History = input.History[len(input.History)-maxHistoryEntries:]
		}
		for i, h := range input.History {
			if len([]rune(h["content"])) > maxHistoryEntryChars {
				truncated := []rune(h["content"])[:maxHistoryEntryChars]
				h["content"] = string(truncated)
				input.History[i] = h
			}
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
			msgRunes := []rune(input.Message)
			titleLen := 30
			if len(msgRunes) < titleLen {
				titleLen = len(msgRunes)
			}
			thread = models.ChatThread{
				UserID:    uid,
				Title:     string(msgRunes[:titleLen]),
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

ПРАВИЛА И ИНТЕРВЬЮ ДЛЯ ROADMAP (Карьерного пути) И ПРОФОРИЕНТАЦИИ:
1. Если пользователь просит провести профориентационную консультацию, составить карьерный путь (Roadmap) или нажать кнопку "Консультация":
   - Проведи глубокую, профессиональную профориентацию. Не давай готовые ответы сразу!
   - Задавай по очереди по 1-2 точных, глубоких вопроса. Расспроси о любимых школьных предметах (точных, гуманитарных или естественных), о его хобби, сильных сторонах, готовности общаться с людьми или работать индивидуально за компьютером, целях по заработку, оценках ЕНТ и желаемом городе.
   - Сделай этот процесс интерактивным и поддерживающим, как сессия с реальным карьерным консультантом. ИИ чат должен полностью знать контекст и помнить все о пользователе.
2. Когда вы определились с профессией или если пользователь прямо запрашивает детальный Roadmap поступления и обучения:
   - Сгенерируй ОЧЕНЬ ПОДРОБНЫЙ, ДЛИННЫЙ И БОЛЬШОЙ роадмап из 6-9 шагов.
   - Каждый шаг в "roadmap" ОБЯЗАТЕЛЬНО должен быть детальным и исчерпывающим: пиши конкретные книги, технологии, фреймворки, пет-проекты и ресурсы.
   - ОБЯЗАТЕЛЬНО включи в роадмап следующие разделы (в виде отдельных первых шагов):
     * Шаг "Что нужно и что готовить (Требования)": Опиши профильные предметы ЕНТ, необходимые баллы, требуемые документы (аттестаты, сертификаты IELTS/SAT) и требования к портфолио.
     * Шаг "Как поступать (Выбор вузов и гранты)": Перечисли лучшие вузы Казахстана (например, NU, KBTU, AITU, SDU, AlmaU) или мира по этой специальности, дедлайны подачи и условия получения государственных и внутренних грантов.
     * Последующие шаги "Что учить и что изучать (Образовательный трек)": Пошаговый технический и практический план освоения профессии от основ до трудоустройства.
3. Возвращай "roadmap" в JSON-ответе только тогда, когда вы определились с профессией и составили план.

ДАННЫЕ ПОЛЬЗОВАТЕЛЯ:
- Имя: %s
- Класс: %s
- Город: %s

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
		currentDate, input.UserInfo.Name, input.UserInfo.Grade, input.UserInfo.City, portfolioStatus)

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

func UpdatePortfolioRoadmap(db *gorm.DB) gin.HandlerFunc {
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
			Roadmap []RoadmapStep `json:"roadmap"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var portfolio models.Portfolio
		if err := db.Where("user_id = ?", uid).First(&portfolio).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Portfolio not found"})
			return
		}

		var aiRes AnalysisResult
		if len(portfolio.AIResult) > 0 {
			if err := json.Unmarshal(portfolio.AIResult, &aiRes); err != nil {
				log.Printf("Unmarshal portfolio.AIResult error: %v", err)
			}
		}

		// Replace the roadmap with the new custom one
		aiRes.Roadmap = input.Roadmap

		newJSON, err := json.Marshal(aiRes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal updated roadmap"})
			return
		}

		portfolio.AIResult = newJSON
		portfolio.UpdatedAt = time.Now()
		if err := db.Save(&portfolio).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save updated portfolio"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "ai_result": aiRes})
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
