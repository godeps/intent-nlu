package intentnlu

import "strings"

var defaultIntentAliases = map[string]string{
	"unknown":                "unknown",
	"calendar":               "calendar_info",
	"date_info":              "calendar_info",
	"holiday_info":           "calendar_info",
	"weather":                "weather_info",
	"forecast":               "weather_info",
	"weather_forecast":       "weather_info",
	"greeting":               "chitchat_greeting",
	"greetings":              "chitchat_greeting",
	"chitchat_greetings":     "chitchat_greeting",
	"chitchat_greeting":      "chitchat_greeting",
	"chitchat_conversations": "chitchat_general",
	"chitchat_botprofile":    "chitchat_general",
	"chitchat_emotion":       "chitchat_general",
	"chitchat_food":          "chitchat_general",
	"chitchat_gossip":        "chitchat_general",
	"chitchat_history":       "chitchat_general",
	"chitchat_humor":         "chitchat_general",
	"chitchat_literature":    "chitchat_general",
	"chitchat_money":         "chitchat_general",
	"chitchat_movies":        "chitchat_general",
	"chitchat_politics":      "chitchat_general",
	"chitchat_psychology":    "chitchat_general",
	"chitchat_science":       "chitchat_general",
	"chitchat_sports":        "chitchat_general",
	"chitchat_trivia":        "chitchat_general",
	"chitchat_ai":            "chitchat_general",
	"chitchat_coding":        "chitchat_general",
	"chitchat_computers":     "chitchat_general",
	"chitchat_health":        "chitchat_general",
	"chitchat_tech_support":  "chitchat_general",
	"chat":                   "chitchat_general",
	"chitchat":               "chitchat_general",
	"video_production":       "creative_video",
	"video_editing":          "creative_video",
	"film_production":        "creative_video",
	"ad_production":          "creative_video",
	"image_generation":       "creative_image",
	"poster_design":          "creative_image",
	"music_creation":         "creative_audio",
	"audio_production":       "creative_audio",
	"3d_modeling":            "creative_3d",
	"video_analysis":         "media_analysis",
	"image_analysis":         "media_analysis",
	// Tool-routing intents
	"search":              "web_search",
	"internet_search":     "web_search",
	"lookup":              "web_search",
	"find_info":           "web_search",
	"google":              "web_search",
	"browse":              "web_search",
	"code":                "coding_assist",
	"programming":         "coding_assist",
	"debug":               "coding_assist",
	"fix_code":            "coding_assist",
	"write_code":          "coding_assist",
	"develop":             "coding_assist",
	"script":              "coding_assist",
	"compile":             "coding_assist",
	"task":                "task_management",
	"todo":                "task_management",
	"kanban":              "task_management",
	"project_management":  "task_management",
	"assign_task":         "task_management",
	"ticket":              "task_management",
	"download":            "file_operation",
	"upload":              "file_operation",
	"read_file":           "file_operation",
	"write_file":          "file_operation",
	"save_file":           "file_operation",
	"fetch_file":          "file_operation",
	"open_file":           "file_operation",
	"recall":              "knowledge_qa",
	"remember":            "knowledge_qa",
	"memory_query":        "knowledge_qa",
	"past_decision":       "knowledge_qa",
	"schedule":            "workflow_automation",
	"cron":                "workflow_automation",
	"automate":            "workflow_automation",
	"pipeline":            "workflow_automation",
	"recurring_task":      "workflow_automation",
	"trigger":             "workflow_automation",
	"analyze_data":        "data_analysis",
	"statistics":          "data_analysis",
	"metrics":             "data_analysis",
	"aggregate":           "data_analysis",
	"report_data":         "data_analysis",
	"create_doc":          "document_creation",
	"write_doc":           "document_creation",
	"readme":              "document_creation",
	"documentation":       "document_creation",
	"guide":               "document_creation",
	"runbook":             "document_creation",
	"translate":           "translation",
	"convert_language":    "translation",
	"localize":            "translation",
	"i18n":                "translation",
	"summarize":           "summarization",
	"tldr":                "summarization",
	"digest":              "summarization",
	"brief":               "summarization",
	"recap":               "summarization",
	"overview":            "summarization",
}

// defaultIntentAliasesRef returns read-only reference to the singleton. Internal use only.
func defaultIntentAliasesRef() map[string]string {
	return defaultIntentAliases
}

// DefaultIntentAliases returns a copy of the stable intent taxonomy aliases.
func DefaultIntentAliases() map[string]string {
	out := make(map[string]string, len(defaultIntentAliases))
	for k, v := range defaultIntentAliases {
		out[k] = v
	}
	return out
}

// NormalizeIntent normalizes one intent to canonical taxonomy label.
func NormalizeIntent(intent string, aliases map[string]string) string {
	intent = strings.ToLower(strings.TrimSpace(intent))
	if intent == "" {
		return ""
	}
	if len(aliases) == 0 {
		return intent
	}
	if canonical, ok := aliases[intent]; ok {
		canonical = strings.ToLower(strings.TrimSpace(canonical))
		if canonical != "" {
			return canonical
		}
	}
	return intent
}

// NormalizeThresholds canonicalizes threshold keys with taxonomy aliases.
func NormalizeThresholds(thresholds map[string]float64, aliases map[string]string) map[string]float64 {
	if len(thresholds) == 0 {
		return nil
	}
	out := make(map[string]float64)
	for key, value := range thresholds {
		if value <= 0 {
			continue
		}
		intent := NormalizeIntent(key, aliases)
		if intent == "" {
			continue
		}
		out[intent] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeIntentAliases(base map[string]string, ext map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(ext))
	for k, v := range base {
		key := strings.ToLower(strings.TrimSpace(k))
		val := strings.ToLower(strings.TrimSpace(v))
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	for k, v := range ext {
		key := strings.ToLower(strings.TrimSpace(k))
		val := strings.ToLower(strings.TrimSpace(v))
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func canonicalIntentsFromClasses(classes []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(classes))
	for _, intent := range classes {
		intent = strings.TrimSpace(intent)
		if intent == "" {
			continue
		}
		if _, ok := seen[intent]; ok {
			continue
		}
		seen[intent] = struct{}{}
		result = append(result, intent)
	}
	return result
}
