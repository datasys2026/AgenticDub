package service

import (
	"encoding/json"
	"fmt"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/util"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

const translationAuditFileNamePattern = "translation_audit_%d.json"

var protectedTermPattern = regexp.MustCompile(`\b(?:[A-Z][A-Za-z0-9]*(?:[-/][A-Za-z0-9]+)*|[A-Za-z]+[0-9]+[A-Za-z0-9-]*|[A-Za-z]*[A-Z][A-Za-z]*[A-Z][A-Za-z0-9-]*)\b`)

type translationAuditRequestItem struct {
	Index           int      `json:"index"`
	Source          string   `json:"source"`
	Translation     string   `json:"translation"`
	ProtectedTerms  []string `json:"protected_terms,omitempty"`
	PreviousSource  string   `json:"previous_source,omitempty"`
	FollowingSource string   `json:"following_source,omitempty"`
}

type translationAuditResponse struct {
	Items []translationAuditResult `json:"items"`
}

type translationAuditResult struct {
	Index               int      `json:"index"`
	Complete            bool     `json:"complete"`
	MissingMeaning      []string `json:"missing_meaning"`
	ProtectedTerms      []string `json:"protected_terms"`
	ShouldRepair        bool     `json:"should_repair"`
	RepairedTranslation string   `json:"repaired_translation"`
}

type translationAuditLog struct {
	RequestItems []translationAuditRequestItem `json:"request_items"`
	Results      []translationAuditResult      `json:"results"`
	RawResponse  string                        `json:"raw_response,omitempty"`
	Error        string                        `json:"error,omitempty"`
}

func (s Service) auditAndRepairTranslations(basePath string, items []*TranslatedItem, targetLang types.StandardLanguageCode, segmentID int) {
	if s.ChatCompleter == nil || len(items) == 0 {
		return
	}

	requestItems := buildTranslationAuditRequestItems(items)
	if len(requestItems) == 0 {
		return
	}

	payload, err := json.MarshalIndent(requestItems, "", "  ")
	if err != nil {
		log.GetLogger().Warn("translation audit marshal request failed", zap.Error(err))
		return
	}

	prompt := fmt.Sprintf(types.TranslationAuditPrompt, types.GetStandardLanguageName(targetLang), string(payload))
	response, err := s.ChatCompleter.ChatCompletion(prompt)
	auditLog := translationAuditLog{
		RequestItems: requestItems,
		RawResponse:  response,
	}
	if err != nil {
		auditLog.Error = err.Error()
		writeTranslationAuditLog(basePath, segmentID, auditLog)
		log.GetLogger().Warn("translation audit failed; continuing without repair", zap.Int("segment", segmentID), zap.Error(err))
		return
	}

	results, err := parseTranslationAuditResponse(response)
	if err != nil {
		auditLog.Error = err.Error()
		writeTranslationAuditLog(basePath, segmentID, auditLog)
		log.GetLogger().Warn("translation audit parse failed; continuing without repair", zap.Int("segment", segmentID), zap.Error(err))
		return
	}
	auditLog.Results = results

	for _, result := range results {
		if result.Index < 0 || result.Index >= len(items) || items[result.Index] == nil {
			continue
		}
		repaired := strings.TrimSpace(result.RepairedTranslation)
		if result.ShouldRepair && repaired != "" {
			if unsafeTranslationAuditRepair(requestItems, result, repaired) {
				log.GetLogger().Warn("translation audit skipped unsafe repair",
					zap.Int("segment", segmentID),
					zap.Int("index", result.Index),
					zap.String("origin", items[result.Index].OriginText),
					zap.String("before", items[result.Index].TranslatedText),
					zap.String("after", repaired))
				continue
			}
			log.GetLogger().Info("translation audit repaired subtitle",
				zap.Int("segment", segmentID),
				zap.Int("index", result.Index),
				zap.String("origin", items[result.Index].OriginText),
				zap.String("before", items[result.Index].TranslatedText),
				zap.String("after", repaired))
			items[result.Index].TranslatedText = repaired
		}
	}

	writeTranslationAuditLog(basePath, segmentID, auditLog)
}

func buildTranslationAuditRequestItems(items []*TranslatedItem) []translationAuditRequestItem {
	requestItems := make([]translationAuditRequestItem, 0, len(items))
	for i, item := range items {
		if item == nil {
			continue
		}
		source := strings.TrimSpace(item.OriginText)
		translation := strings.TrimSpace(item.TranslatedText)
		if source == "" || translation == "" {
			continue
		}

		requestItem := translationAuditRequestItem{
			Index:          i,
			Source:         source,
			Translation:    translation,
			ProtectedTerms: extractProtectedTerms(source),
		}
		if i > 0 && items[i-1] != nil {
			requestItem.PreviousSource = strings.TrimSpace(items[i-1].OriginText)
		}
		if i < len(items)-1 && items[i+1] != nil {
			requestItem.FollowingSource = strings.TrimSpace(items[i+1].OriginText)
		}
		requestItems = append(requestItems, requestItem)
	}
	return requestItems
}

func unsafeTranslationAuditRepair(requestItems []translationAuditRequestItem, result translationAuditResult, repaired string) bool {
	var request *translationAuditRequestItem
	for i := range requestItems {
		if requestItems[i].Index == result.Index {
			request = &requestItems[i]
			break
		}
	}
	if request == nil {
		return false
	}

	sourceWords := strings.Fields(request.Source)
	if len(sourceWords) <= 4 && cjkRuneCount(repaired) > 8 {
		return true
	}

	sourceLower := strings.ToLower(request.Source)
	previousLower := strings.ToLower(request.PreviousSource)
	followingLower := strings.ToLower(request.FollowingSource)
	for _, missing := range result.MissingMeaning {
		missingLower := strings.ToLower(strings.TrimSpace(missing))
		if missingLower == "" || strings.Contains(sourceLower, missingLower) {
			continue
		}
		if strings.Contains(previousLower, missingLower) || strings.Contains(followingLower, missingLower) {
			return true
		}
	}
	return false
}

func parseTranslationAuditResponse(response string) ([]translationAuditResult, error) {
	cleaned := util.CleanMarkdownCodeBlock(strings.TrimSpace(response))
	var parsed translationAuditResponse
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		var direct []translationAuditResult
		if directErr := json.Unmarshal([]byte(cleaned), &direct); directErr != nil {
			return nil, err
		}
		parsed.Items = direct
	}
	return parsed.Items, nil
}

func extractProtectedTerms(source string) []string {
	matches := protectedTermPattern.FindAllString(source, -1)
	seen := make(map[string]bool, len(matches))
	terms := make([]string, 0, len(matches))
	for _, match := range matches {
		match = strings.Trim(match, ".,!?;:()[]{}\"'")
		if match == "" || isCommonCapitalizedWord(match) || seen[match] {
			continue
		}
		seen[match] = true
		terms = append(terms, match)
	}
	return terms
}

func isCommonCapitalizedWord(term string) bool {
	common := map[string]bool{
		"I": true, "A": true, "The": true, "This": true, "That": true, "These": true, "Those": true,
		"And": true, "But": true, "So": true, "Because": true, "When": true, "While": true,
		"If": true, "It": true, "ItS": true, "We": true, "You": true, "He": true, "She": true,
		"They": true, "There": true, "Here": true, "Let": true, "Oh": true, "Wow": true,
	}
	return common[term]
}

func writeTranslationAuditLog(basePath string, segmentID int, auditLog translationAuditLog) {
	if basePath == "" {
		return
	}
	data, err := json.MarshalIndent(auditLog, "", "  ")
	if err != nil {
		log.GetLogger().Warn("translation audit marshal log failed", zap.Error(err))
		return
	}
	path := filepath.Join(basePath, fmt.Sprintf(translationAuditFileNamePattern, segmentID))
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.GetLogger().Warn("translation audit write log failed", zap.String("path", path), zap.Error(err))
	}
}
