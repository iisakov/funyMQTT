package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// DecodeMessage пытается декодировать сообщение различными способами
func DecodeMessage(payload []byte) (string, error) {
	// Попытка 1: Просто как строка (если это обычный текст)
	text := string(payload)
	if isPrintable(text) {
		return text, nil
	}

	// Попытка 2: Base64 декодирование
	if decoded, err := base64.StdEncoding.DecodeString(text); err == nil {
		decodedText := string(decoded)
		if isPrintable(decodedText) {
			return decodedText, nil
		}
	}

	// Попытка 3: Base64 URL декодирование
	if decoded, err := base64.URLEncoding.DecodeString(text); err == nil {
		decodedText := string(decoded)
		if isPrintable(decodedText) {
			return decodedText, nil
		}
	}

	// Попытка 4: JSON форматирование
	if json.Valid(payload) {
		var jsonData interface{}
		if err := json.Unmarshal(payload, &jsonData); err == nil {
			if prettyJSON, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
				return string(prettyJSON), nil
			}
		}
	}

	// Если ничего не помогло, возвращаем как hex или base64
	return fmt.Sprintf("Бинарные данные [%d байт], base64: %s", len(payload),
		base64.StdEncoding.EncodeToString(payload)), nil
}

// isPrintable проверяет, состоит ли строка в основном из печатных символов
func isPrintable(s string) bool {
	if len(s) == 0 {
		return false
	}

	printableCount := 0
	for _, r := range s {
		if r >= 32 && r <= 126 || r == '\n' || r == '\r' || r == '\t' {
			printableCount++
		}
	}

	// Считаем строку печатной если хотя бы 80% символов печатные
	return float64(printableCount)/float64(len(s)) >= 0.8
}
