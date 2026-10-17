package insight

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// Límites del diagnóstico. Lo que los supera se recorta o se descarta: el
// modelo no siempre respeta las indicaciones de longitud.
const (
	MaxItems         = 4
	maxHeadlineLen   = 140
	maxSummaryLen    = 700
	maxInsightLen    = 300
	maxTitleLen      = 80
	maxDetailLen     = 300
	maxSummaryBreaks = 2
)

// ErrInvalidDiagnosis indica que la respuesta del modelo no contiene un diagnóstico utilizable.
var ErrInvalidDiagnosis = errors.New("el diagnóstico generado está vacío o es inválido")

// Diagnosis es el análisis personalizado que genera la IA, como texto plano.
type Diagnosis struct {
	// Headline resume el diagnóstico en una frase.
	Headline string
	// Summary lo desarrolla en uno o dos párrafos.
	Summary         string
	FinanceInsights []string
	HabitInsights   []string
	Suggestions     []Suggestion
}

// Suggestion es una acción concreta que se recomienda al usuario.
type Suggestion struct {
	Title  string
	Detail string
}

// markdownPattern son marcas de formato que el modelo a veces añade aunque se
// le pida texto plano: énfasis, código y viñetas o encabezados al inicio.
var markdownPattern = regexp.MustCompile("\\*\\*|__|`|(?m)^\\s*(?:[-*•]|#{1,6})\\s+")

// Sanitized devuelve una copia de d apta para guardar o mostrar: texto plano
// sin caracteres de control ni formato Markdown, recortado a los límites y sin
// elementos vacíos. Devuelve ErrInvalidDiagnosis si no queda un titular ni un
// resumen.
func (d Diagnosis) Sanitized() (Diagnosis, error) {
	clean := Diagnosis{
		Headline:        cleanLine(d.Headline, maxHeadlineLen),
		Summary:         cleanParagraphs(d.Summary, maxSummaryLen),
		FinanceInsights: cleanList(d.FinanceInsights, maxInsightLen),
		HabitInsights:   cleanList(d.HabitInsights, maxInsightLen),
	}
	for _, s := range d.Suggestions {
		if len(clean.Suggestions) == MaxItems {
			break
		}
		title, detail := cleanLine(s.Title, maxTitleLen), cleanLine(s.Detail, maxDetailLen)
		if title == "" && detail == "" {
			continue
		}
		clean.Suggestions = append(clean.Suggestions, Suggestion{Title: title, Detail: detail})
	}
	if clean.Headline == "" && clean.Summary == "" {
		return Diagnosis{}, ErrInvalidDiagnosis
	}
	return clean, nil
}

// cleanLine deja s en una sola línea de texto plano de hasta max runas.
func cleanLine(s string, max int) string {
	s = stripControl(markdownPattern.ReplaceAllString(s, ""), false)
	return truncate(strings.Join(strings.Fields(s), " "), max)
}

// cleanParagraphs es como cleanLine pero conserva hasta maxSummaryBreaks
// saltos de párrafo.
func cleanParagraphs(s string, max int) string {
	s = stripControl(markdownPattern.ReplaceAllString(s, ""), true)
	var paragraphs []string
	for _, p := range strings.Split(s, "\n") {
		if p = strings.Join(strings.Fields(p), " "); p != "" {
			paragraphs = append(paragraphs, p)
		}
	}
	if len(paragraphs) > maxSummaryBreaks+1 {
		last := strings.Join(paragraphs[maxSummaryBreaks:], " ")
		paragraphs = append(paragraphs[:maxSummaryBreaks], last)
	}
	return truncate(strings.Join(paragraphs, "\n\n"), max)
}

func cleanList(items []string, max int) []string {
	var out []string
	for _, item := range items {
		if len(out) == MaxItems {
			break
		}
		if item = cleanLine(item, max); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// stripControl reemplaza por espacios los caracteres de control y de formato
// invisibles (p. ej. marcas de dirección de texto). Con keepNewlines conserva "\n".
func stripControl(s string, keepNewlines bool) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' && keepNewlines:
			return r
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			return ' '
		}
		return r
	}, s)
}
