package service

import (
	"encoding/json"
	"fmt"
	"krillin-ai/internal/agent/hitl"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

const (
	reviewAuditFileName     = "review_audit.json"
	reviewSuggestedFileName = "review_suggested.txt"
)

type reviewAuditLog struct {
	RequestItems []translationAuditRequestItem `json:"request_items"`
	Results      []reviewAuditResult           `json:"results"`
	RawResponse  string                        `json:"raw_response,omitempty"`
	Error        string                        `json:"error,omitempty"`
}

type reviewAuditResult struct {
	Index               int      `json:"index"`
	SegmentIndex        int      `json:"segment_index"`
	Complete            bool     `json:"complete"`
	MissingMeaning      []string `json:"missing_meaning"`
	ProtectedTerms      []string `json:"protected_terms"`
	ShouldRepair        bool     `json:"should_repair"`
	RepairedTranslation string   `json:"repaired_translation"`
	PreviousSubtitle    string   `json:"previous_subtitle"`
	AppliedSubtitle     string   `json:"applied_subtitle"`
	Applied             bool     `json:"applied"`
}

func (s Service) auditAndSuggestReview(basePath string, doc hitl.ReviewDocument, targetLang types.StandardLanguageCode) hitl.ReviewDocument {
	if s.ChatCompleter == nil || len(doc.Segments) == 0 {
		return doc
	}

	requestItems := buildReviewAuditRequestItems(doc)
	if len(requestItems) == 0 {
		return doc
	}

	payload, err := json.MarshalIndent(requestItems, "", "  ")
	if err != nil {
		log.GetLogger().Warn("review audit marshal request failed", zap.Error(err))
		return doc
	}

	prompt := fmt.Sprintf(types.TranslationAuditPrompt, types.GetStandardLanguageName(targetLang), string(payload))
	response, err := s.ChatCompleter.ChatCompletion(prompt)
	auditLog := reviewAuditLog{
		RequestItems: requestItems,
		RawResponse:  response,
	}
	if err != nil {
		auditLog.Error = err.Error()
		writeReviewAuditLog(basePath, auditLog)
		log.GetLogger().Warn("review audit failed; continuing without suggestions", zap.Error(err))
		return doc
	}

	results, err := parseTranslationAuditResponse(response)
	if err != nil {
		auditLog.Error = err.Error()
		writeReviewAuditLog(basePath, auditLog)
		log.GetLogger().Warn("review audit parse failed; continuing without suggestions", zap.Error(err))
		return doc
	}

	auditLog.Results = applyReviewAuditResults(&doc, results)
	writeReviewAuditLog(basePath, auditLog)
	writeSuggestedReview(basePath, doc)
	return doc
}

func buildReviewAuditRequestItems(doc hitl.ReviewDocument) []translationAuditRequestItem {
	requestItems := make([]translationAuditRequestItem, 0, len(doc.Segments))
	for i, segment := range doc.Segments {
		source := strings.TrimSpace(segment.Original)
		if source == "" {
			continue
		}

		requestItem := translationAuditRequestItem{
			Index:          i,
			Source:         source,
			Translation:    strings.TrimSpace(segment.Edited),
			ProtectedTerms: extractProtectedTerms(source),
		}
		if i > 0 {
			requestItem.PreviousSource = strings.TrimSpace(doc.Segments[i-1].Original)
		}
		if i < len(doc.Segments)-1 {
			requestItem.FollowingSource = strings.TrimSpace(doc.Segments[i+1].Original)
		}
		requestItems = append(requestItems, requestItem)
	}
	return requestItems
}

func applyReviewAuditResults(doc *hitl.ReviewDocument, results []translationAuditResult) []reviewAuditResult {
	appliedResults := make([]reviewAuditResult, 0, len(results))
	for _, result := range results {
		auditResult := reviewAuditResult{
			Index:               result.Index,
			Complete:            result.Complete,
			MissingMeaning:      result.MissingMeaning,
			ProtectedTerms:      result.ProtectedTerms,
			ShouldRepair:        result.ShouldRepair,
			RepairedTranslation: result.RepairedTranslation,
		}
		if result.Index < 0 || result.Index >= len(doc.Segments) {
			appliedResults = append(appliedResults, auditResult)
			continue
		}

		segment := &doc.Segments[result.Index]
		auditResult.SegmentIndex = segment.Index
		auditResult.PreviousSubtitle = segment.Edited

		repaired := hitl.CleanPunctuation(strings.TrimSpace(result.RepairedTranslation))
		if result.ShouldRepair && repaired != "" {
			segment.Edited = repaired
			auditResult.AppliedSubtitle = repaired
			auditResult.Applied = true
		}
		appliedResults = append(appliedResults, auditResult)
	}
	return appliedResults
}

func writeReviewAuditLog(basePath string, auditLog reviewAuditLog) {
	if basePath == "" {
		return
	}
	data, err := json.MarshalIndent(auditLog, "", "  ")
	if err != nil {
		log.GetLogger().Warn("review audit marshal log failed", zap.Error(err))
		return
	}
	path := filepath.Join(basePath, reviewAuditFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.GetLogger().Warn("review audit write log failed", zap.String("path", path), zap.Error(err))
	}
}

func writeSuggestedReview(basePath string, doc hitl.ReviewDocument) {
	if basePath == "" {
		return
	}
	content, err := hitl.TxtParser{}.Generate(doc)
	if err != nil {
		log.GetLogger().Warn("review audit generate suggested review failed", zap.Error(err))
		return
	}
	path := filepath.Join(basePath, reviewSuggestedFileName)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		log.GetLogger().Warn("review audit write suggested review failed", zap.String("path", path), zap.Error(err))
	}
}
