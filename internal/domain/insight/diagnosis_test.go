package insight

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDiagnosisSanitized(t *testing.T) {
	tests := []struct {
		name    string
		in      Diagnosis
		want    Diagnosis
		wantErr error
	}{
		{
			name: "quita Markdown, controles y espacios sobrantes",
			in: Diagnosis{
				Headline:        "  **Buen mes**\tpara tu ahorro​ ",
				Summary:         "Primer párrafo.\n\n\n- Segundo   párrafo con `código`.\x07",
				FinanceInsights: []string{"* Gastas 30 % en comida", "   ", "## Ahorro del 25 %"},
				HabitInsights:   []string{"__Meditar__ va en racha de 3 días"},
				Suggestions: []Suggestion{
					{Title: " Fija un tope ", Detail: "Comida: 120 000 CRC\nal mes"},
					{Title: "", Detail: " "},
				},
			},
			want: Diagnosis{
				Headline:        "Buen mes para tu ahorro",
				Summary:         "Primer párrafo.\n\nSegundo párrafo con código.",
				FinanceInsights: []string{"Gastas 30 % en comida", "Ahorro del 25 %"},
				HabitInsights:   []string{"Meditar va en racha de 3 días"},
				Suggestions:     []Suggestion{{Title: "Fija un tope", Detail: "Comida: 120 000 CRC al mes"}},
			},
		},
		{
			name: "recorta listas y textos largos",
			in: Diagnosis{
				Headline:        strings.Repeat("a", 500),
				FinanceInsights: []string{"1", "2", "3", "4", "5", "6"},
				Suggestions:     []Suggestion{{Title: "1"}, {Title: "2"}, {Title: "3"}, {Title: "4"}, {Title: "5"}},
			},
			want: Diagnosis{
				Headline:        strings.Repeat("a", maxHeadlineLen-1) + "…",
				FinanceInsights: []string{"1", "2", "3", "4"},
				Suggestions:     []Suggestion{{Title: "1"}, {Title: "2"}, {Title: "3"}, {Title: "4"}},
			},
		},
		{
			name: "une los párrafos que sobran",
			in:   Diagnosis{Summary: "uno\ndos\ntres\ncuatro"},
			want: Diagnosis{Summary: "uno\n\ndos\n\ntres cuatro"},
		},
		{
			name:    "sin titular ni resumen",
			in:      Diagnosis{Headline: " ** ", FinanceInsights: []string{"algo"}},
			wantErr: ErrInvalidDiagnosis,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Sanitized()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sanitized() error = %v, se esperaba %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sanitized() =\n%#v\nse esperaba\n%#v", got, tt.want)
			}
		})
	}
}
