package service

import (
	"strings"
	"titip-jejak-api/internal/model"
)

const MinMatchScore = 60

// ScoreReports menghitung skor kecocokan antara found report dan missing report.
// Scoring: lokasi(40) + gender(30) + usia(20) + deskripsi(10) = max 100
func ScoreReports(found, missing model.Report) int {
	score := 0
	score += scoreLocation(found, missing)
	score += scoreGender(found, missing)
	score += scoreAge(found, missing)
	score += scoreDescription(found, missing)
	return score
}

// scoreLocation — max 40 poin
// same city = 40, same province = 20, beda = 0
func scoreLocation(found, missing model.Report) int {
	if strings.EqualFold(strings.TrimSpace(found.City), strings.TrimSpace(missing.City)) {
		return 40
	}
	if strings.EqualFold(strings.TrimSpace(found.Province), strings.TrimSpace(missing.Province)) {
		return 20
	}
	return 0
}

// scoreGender — max 30 poin
// cocok = 30, salah satu unknown = 15, beda = 0
func scoreGender(found, missing model.Report) int {
	f := strings.ToLower(string(found.Gender))
	m := strings.ToLower(string(missing.Gender))

	if f == "unknown" || m == "unknown" {
		return 15
	}
	if f == m {
		return 30
	}
	return 0
}

// scoreAge — max 20 poin
// salah satu nil = 0
// selisih ≤5  = 20, ≤10 = 10, >10 = 0
func scoreAge(found, missing model.Report) int {
	if found.EstimatedAge == nil || missing.EstimatedAge == nil {
		return 0
	}

	diff := *found.EstimatedAge - *missing.EstimatedAge
	if diff < 0 {
		diff = -diff
	}

	switch {
	case diff <= 5:
		return 20
	case diff <= 10:
		return 10
	default:
		return 0
	}
}

// scoreDescription — max 10 poin
// hitung overlap kata (stopword-minimal) antara dua deskripsi
func scoreDescription(found, missing model.Report) int {
	foundWords := tokenize(found.Description)
	missingWords := tokenize(missing.Description)

	if len(foundWords) == 0 || len(missingWords) == 0 {
		return 0
	}

	// Buat set dari kata found
	foundSet := make(map[string]struct{}, len(foundWords))
	for _, w := range foundWords {
		foundSet[w] = struct{}{}
	}

	// Hitung irisan
	overlap := 0
	for _, w := range missingWords {
		if _, ok := foundSet[w]; ok {
			overlap++
		}
	}

	// Rasio overlap terhadap total kata unik missing
	ratio := float64(overlap) / float64(len(missingWords))
	switch {
	case ratio >= 0.3:
		return 10
	case ratio >= 0.15:
		return 5
	default:
		return 0
	}
}

// tokenize memecah teks menjadi kata-kata lowercase, minimal 3 karakter,
// dan menyaring stopword umum Bahasa Indonesia.
func tokenize(text string) []string {
	stopwords := map[string]struct{}{
		"dan": {}, "yang": {}, "di": {}, "ke": {}, "dari": {},
		"ini": {}, "itu": {}, "dengan": {}, "untuk": {}, "ada": {},
		"tidak": {}, "adalah": {}, "atau": {}, "juga": {}, "pada": {},
		"sudah": {}, "saat": {}, "nya": {}, "ber": {}, "ter": {},
	}

	raw := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})

	result := make([]string, 0, len(raw))
	seen := make(map[string]struct{})
	for _, w := range raw {
		if len(w) < 3 {
			continue
		}
		if _, stop := stopwords[w]; stop {
			continue
		}
		if _, dup := seen[w]; dup {
			continue
		}
		seen[w] = struct{}{}
		result = append(result, w)
	}
	return result
}
