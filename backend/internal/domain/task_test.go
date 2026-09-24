package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Priority
// ============================================================================

func TestPriority_IsValid(t *testing.T) {
	cases := []struct {
		name  string
		input Priority
		want  bool
	}{
		{"LOW es válido", PriorityLow, true},
		{"MEDIUM es válido", PriorityMedium, true},
		{"HIGH es válido", PriorityHigh, true},
		{"minúsculas no son válidas", "low", false},
		{"valor desconocido no es válido", "URGENT", false},
		{"string vacío no es válido", "", false},
		{"espacios no son válidos", " HIGH ", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.IsValid()
			if got != tc.want {
				t.Errorf("Priority(%q).IsValid() = %v, se esperaba %v",
					tc.input, got, tc.want)
			}
		})
	}
}

// ============================================================================
// Task
// ============================================================================

func TestTask_IsDeleted(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name string
		task Task
		want bool
	}{
		{
			name: "sin deleted_at no está eliminada",
			task: Task{DeletedAt: nil},
			want: false,
		},
		{
			name: "con deleted_at está eliminada",
			task: Task{DeletedAt: &now},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.task.IsDeleted(); got != tc.want {
				t.Errorf("IsDeleted() = %v, se esperaba %v", got, tc.want)
			}
		})
	}
}

// ============================================================================
// CreateTaskInput.Validate
// ============================================================================

func TestCreateTaskInput_Validate(t *testing.T) {
	longTitle := strings.Repeat("a", 256)
	maxTitle := strings.Repeat("a", 255)
	longTag := strings.Repeat("t", 51)

	cases := []struct {
		name      string
		input     CreateTaskInput
		wantErr   bool
		wantField string // solo se verifica si wantErr es true
	}{
		{
			name:    "título válido",
			input:   CreateTaskInput{Title: "Hacer ejercicio"},
			wantErr: false,
		},
		{
			name:    "título con espacios al borde",
			input:   CreateTaskInput{Title: "  Hacer ejercicio  "},
			wantErr: false,
		},
		{
			name:    "título exactamente 255 caracteres",
			input:   CreateTaskInput{Title: maxTitle},
			wantErr: false,
		},
		{
			name:      "título vacío",
			input:     CreateTaskInput{Title: ""},
			wantErr:   true,
			wantField: "title",
		},
		{
			name:      "título solo con espacios",
			input:     CreateTaskInput{Title: "   "},
			wantErr:   true,
			wantField: "title",
		},
		{
			name:      "título de 256 caracteres",
			input:     CreateTaskInput{Title: longTitle},
			wantErr:   true,
			wantField: "title",
		},
		{
			name:    "prioridad vacía (default permitido)",
			input:   CreateTaskInput{Title: "X", Priority: ""},
			wantErr: false,
		},
		{
			name:    "prioridad HIGH",
			input:   CreateTaskInput{Title: "X", Priority: PriorityHigh},
			wantErr: false,
		},
		{
			name:      "prioridad inválida",
			input:     CreateTaskInput{Title: "X", Priority: "URGENT"},
			wantErr:   true,
			wantField: "priority",
		},
		{
			name:    "tags válidos",
			input:   CreateTaskInput{Title: "X", TagNames: []string{"trabajo", "urgente"}},
			wantErr: false,
		},
		{
			name:    "lista de tags vacía",
			input:   CreateTaskInput{Title: "X", TagNames: []string{}},
			wantErr: false,
		},
		{
			name:      "tag vacío en la lista",
			input:     CreateTaskInput{Title: "X", TagNames: []string{"trabajo", ""}},
			wantErr:   true,
			wantField: "tag_names",
		},
		{
			name:      "tag con solo espacios",
			input:     CreateTaskInput{Title: "X", TagNames: []string{"   "}},
			wantErr:   true,
			wantField: "tag_names",
		},
		{
			name:      "tag de 51 caracteres",
			input:     CreateTaskInput{Title: "X", TagNames: []string{longTag}},
			wantErr:   true,
			wantField: "tag_names",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("Validate() devolvió error inesperado: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Validate() = nil, se esperaba un error")
			}

			// Si hay un campo esperado, verificamos que el ValidationError
			// apunte a ese campo.
			if tc.wantField != "" {
				var vErr *ValidationError
				if !errors.As(err, &vErr) {
					t.Fatalf("el error no es *ValidationError: %T", err)
				}
				if vErr.Field != tc.wantField {
					t.Errorf("Field = %q, se esperaba %q", vErr.Field, tc.wantField)
				}
			}

			// Todo error de validación debe ser detectable como ErrInvalidInput.
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("errors.Is(err, ErrInvalidInput) = false, se esperaba true")
			}
		})
	}
}

// ============================================================================
// UpdateTaskInput.Validate
// ============================================================================

func TestUpdateTaskInput_Validate(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	prioPtr := func(p Priority) *Priority { return &p }
	emptyStr := ""

	cases := []struct {
		name      string
		input     UpdateTaskInput
		wantErr   bool
		wantField string
	}{
		{
			name:    "todos los campos nil (vacío pero válido)",
			input:   UpdateTaskInput{},
			wantErr: false,
		},
		{
			name:    "solo título válido",
			input:   UpdateTaskInput{Title: strPtr("Nuevo título")},
			wantErr: false,
		},
		{
			name:      "título vacío",
			input:     UpdateTaskInput{Title: &emptyStr},
			wantErr:   true,
			wantField: "title",
		},
		{
			name:      "título solo espacios",
			input:     UpdateTaskInput{Title: strPtr("   ")},
			wantErr:   true,
			wantField: "title",
		},
		{
			name:      "prioridad inválida",
			input:     UpdateTaskInput{Priority: prioPtr("URGENT")},
			wantErr:   true,
			wantField: "priority",
		},
		{
			name:    "prioridad válida",
			input:   UpdateTaskInput{Priority: prioPtr(PriorityHigh)},
			wantErr: false,
		},
		{
			name:      "tag vacío en lista",
			input:     UpdateTaskInput{TagNames: &[]string{""}},
			wantErr:   true,
			wantField: "tag_names",
		},
		{
			name:    "tag válido en lista",
			input:   UpdateTaskInput{TagNames: &[]string{"trabajo"}},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("Validate() devolvió error inesperado: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Validate() = nil, se esperaba un error")
			}

			if tc.wantField != "" {
				var vErr *ValidationError
				if !errors.As(err, &vErr) {
					t.Fatalf("el error no es *ValidationError: %T", err)
				}
				if vErr.Field != tc.wantField {
					t.Errorf("Field = %q, se esperaba %q", vErr.Field, tc.wantField)
				}
			}
		})
	}
}

// ============================================================================
// UpdateTaskInput.IsEmpty
// ============================================================================

func TestUpdateTaskInput_IsEmpty(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	boolPtr := func(b bool) *bool { return &b }
	prioPtr := func(p Priority) *Priority { return &p }

	cases := []struct {
		name  string
		input UpdateTaskInput
		want  bool
	}{
		{
			name:  "todos los campos nil",
			input: UpdateTaskInput{},
			want:  true,
		},
		{
			name:  "solo título",
			input: UpdateTaskInput{Title: strPtr("X")},
			want:  false,
		},
		{
			name:  "solo completed false",
			input: UpdateTaskInput{Completed: boolPtr(false)},
			want:  false,
		},
		{
			name:  "solo prioridad",
			input: UpdateTaskInput{Priority: prioPtr(PriorityLow)},
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.input.IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, se esperaba %v", got, tc.want)
			}
		})
	}
}
