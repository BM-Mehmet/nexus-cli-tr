package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// CleanJSON tries to extract and validate pure JSON from model output that might
// be wrapped in markdown code blocks or have surrounding text.
func CleanJSON(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}", nil
	}

	// 1. Check if already valid JSON
	if json.Valid([]byte(trimmed)) {
		var buf bytes.Buffer
		if err := json.Compact(&buf, []byte(trimmed)); err == nil {
			return buf.String(), nil
		}
		return trimmed, nil
	}

	// 2. Try removing markdown code blocks: ```json ... ``` or ``` ... ```
	if strings.Contains(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		var extracted strings.Builder
		inBlock := false
		for _, line := range lines {
			lineTrimmed := strings.TrimSpace(line)
			if strings.HasPrefix(lineTrimmed, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				extracted.WriteString(line)
				extracted.WriteString("\n")
			}
		}
		candidate := strings.TrimSpace(extracted.String())
		if candidate != "" && json.Valid([]byte(candidate)) {
			return candidate, nil
		}
	}

	// 3. Find outermost { ... } or [ ... ]
	startObj := strings.Index(trimmed, "{")
	endObj := strings.LastIndex(trimmed, "}")
	startArr := strings.Index(trimmed, "[")
	endArr := strings.LastIndex(trimmed, "]")

	// Check object
	if startObj != -1 && endObj != -1 && endObj > startObj {
		objCandidate := strings.TrimSpace(trimmed[startObj : endObj+1])
		if json.Valid([]byte(objCandidate)) {
			return objCandidate, nil
		}
	}

	// Check array
	if startArr != -1 && endArr != -1 && endArr > startArr {
		arrCandidate := strings.TrimSpace(trimmed[startArr : endArr+1])
		if json.Valid([]byte(arrCandidate)) {
			return arrCandidate, nil
		}
	}

	return trimmed, fmt.Errorf("metinde geçerli JSON biçimi bulunamadı")
}

// BuildJSONInstruction creates a strict system prompt instruction for JSON output.
func BuildJSONInstruction(formatRaw json.RawMessage) string {
	if len(formatRaw) == 0 {
		return ""
	}

	rawStr := strings.TrimSpace(string(formatRaw))
	if rawStr == `""` || rawStr == "null" {
		return ""
	}

	var instruction strings.Builder
	instruction.WriteString("\n[ZORUNLU ÇIKTI BİÇİMİ]:\n")
	instruction.WriteString("YALNIZCA geçerli JSON yanıtı verin. Çıktıyı ```json ... ``` gibi Markdown kod bloklarına almayın.\n")
	instruction.WriteString("JSON nesnesinin dışında selamlama, açıklama, düşünce veya önsöz eklemeyin.\n")

	// If formatRaw is literally "json"
	if rawStr == `"json"` || rawStr == `json` {
		instruction.WriteString("Yanıtınızı temsil eden geçerli bir JSON nesnesi veya dizisi üretin.\n")
		return instruction.String()
	}

	// Check if format is a JSON Schema object
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(formatRaw, &schemaMap); err == nil && len(schemaMap) > 0 {
		instruction.WriteString("JSON çıktınız aşağıdaki JSON Schema şemasına kesinlikle uymalıdır:\n")
		schemaBytes, _ := json.MarshalIndent(schemaMap, "", "  ")
		instruction.WriteString(string(schemaBytes))
		instruction.WriteString("\n")
		return instruction.String()
	}

	instruction.WriteString("Kullanıcının istediği yapıya uygun, yalnızca geçerli JSON üretin.\n")
	return instruction.String()
}
