package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"krillin-ai/internal/types"
)

const (
	defaultSpeakerID                  = "narrator"
	ttsVoiceAssignmentsFileName       = "tts_voice_assignments.json"
	defaultTTSInstructionalSlowLength = 28
)

var (
	speakerPrefixRegex = regexp.MustCompile(`^\s*(?:\[([A-Za-z0-9 _-]{1,32})\]|(Speaker\s*[0-9A-Za-z_-]{1,16}|說話者\s*[0-9A-Za-z_-]{1,16}|旁白))\s*[:：]\s*(.+)$`)
	speechTagRegex     = regexp.MustCompile(`(?i)\[(?:pause|long-pause|hum-tune|laugh|chuckle|giggle|cry|tsk|tongue-click|lip-smack|breath|inhale|exhale|sigh)\]|</?(?:soft|whisper|loud|build-intensity|decrease-intensity|higher-pitch|lower-pitch|slow|fast|sing-song|singing|laugh-speak|emphasis)>`)
)

func buildSpeakerVoiceAssignments(subtitles []types.SrtSentenceWithStrTime, defaultVoice string) map[string]string {
	assignments := map[string]string{defaultSpeakerID: defaultVoice}
	for _, subtitle := range subtitles {
		speakerID, _ := splitSpeakerPrefix(subtitle.Text)
		if speakerID == "" {
			speakerID = defaultSpeakerID
		}
		if _, ok := assignments[speakerID]; !ok {
			assignments[speakerID] = defaultVoice
		}
	}
	return assignments
}

func splitSpeakerPrefix(text string) (speakerID, displayText string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultSpeakerID, ""
	}
	matches := speakerPrefixRegex.FindStringSubmatch(text)
	if len(matches) == 0 {
		return defaultSpeakerID, text
	}
	speakerID = strings.TrimSpace(matches[1])
	if speakerID == "" {
		speakerID = strings.TrimSpace(matches[2])
	}
	speakerID = normalizeSpeakerID(speakerID)
	displayText = strings.TrimSpace(matches[3])
	if displayText == "" {
		displayText = text
	}
	return speakerID, displayText
}

func normalizeSpeakerID(speakerID string) string {
	speakerID = strings.ToLower(strings.TrimSpace(speakerID))
	speakerID = strings.Join(strings.Fields(speakerID), "-")
	speakerID = strings.Trim(speakerID, "_-")
	if speakerID == "" {
		return defaultSpeakerID
	}
	return speakerID
}

func buildTTSSpeechText(displayText string) string {
	text := strings.TrimSpace(displayText)
	if text == "" {
		return text
	}
	if shouldSpeakSlowly(text) && !strings.Contains(strings.ToLower(text), "<slow>") {
		return "<slow>" + text + "</slow>"
	}
	return text
}

func shouldSpeakSlowly(text string) bool {
	if len([]rune(text)) < defaultTTSInstructionalSlowLength {
		return false
	}
	for _, token := range []string{"API", "AI", "UI", "UX", "LLM", "TTS", "STT", "OpenAI", "xAI", "Grok"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func stripSpeechTagsFromDisplay(text string) string {
	return speechTagRegex.ReplaceAllString(text, "")
}

func writeSpeakerVoiceAssignments(taskBasePath string, assignments map[string]string) error {
	if len(assignments) == 0 {
		return nil
	}
	rows := make([]speakerVoiceAssignment, 0, len(assignments))
	for speakerID, voiceID := range assignments {
		rows = append(rows, speakerVoiceAssignment{SpeakerID: speakerID, VoiceID: voiceID})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].SpeakerID < rows[j].SpeakerID
	})
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(taskBasePath, ttsVoiceAssignmentsFileName), append(data, '\n'), 0644)
}
