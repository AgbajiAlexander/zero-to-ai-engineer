package curriculum

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultLoadsComputationalThinkingCurriculum(t *testing.T) {
	got, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if got.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", got.Version, "1.0.0")
	}
	if len(got.Skills) != 4 {
		t.Errorf("skills = %d, want 4", len(got.Skills))
	}
	if len(got.Milestones) != 1 {
		t.Errorf("milestones = %d, want 1", len(got.Milestones))
	}
	if err := got.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestLoadRejectsMalformedDocuments(t *testing.T) {
	tests := []struct {
		name     string
		document string
	}{
		{
			name:     "malformed JSON",
			document: `{"schema_version":`,
		},
		{
			name: "unknown field",
			document: `{"schema_version":1,"version":"1.0.0","title":"Test",` +
				`"skills":[],"milestones":[],"activities":[],"unexpected":true}`,
		},
		{
			name:     "multiple JSON values",
			document: `{} {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load([]byte(tt.document))
			if !errors.Is(err, ErrInvalidCurriculum) {
				t.Errorf("Load() error = %v, want ErrInvalidCurriculum", err)
			}
		})
	}
}

func TestValidateRejectsUnsupportedSchema(t *testing.T) {
	value := validFixture()
	value.SchemaVersion = supportedSchemaVersion + 1
	assertInvalid(t, value, "unsupported schema")
}

func TestValidateRejectsDuplicateSkillIDs(t *testing.T) {
	value := validFixture()
	value.Skills[1].ID = value.Skills[0].ID
	assertInvalid(t, value, "duplicate skill")
}

func TestValidateRejectsMissingPrerequisite(t *testing.T) {
	value := validFixture()
	value.Skills[1].Prerequisites = []string{"missing-skill"}
	assertInvalid(t, value, "missing prerequisite")
}

func TestValidateRejectsSelfPrerequisite(t *testing.T) {
	value := validFixture()
	value.Skills[0].Prerequisites = []string{value.Skills[0].ID}
	assertInvalid(t, value, "cannot depend on itself")
}

func TestValidateRejectsPrerequisiteCycle(t *testing.T) {
	value := validFixture()
	value.Skills[0].Prerequisites = []string{value.Skills[1].ID}
	value.Skills[1].Prerequisites = []string{value.Skills[0].ID}
	assertInvalid(t, value, "cycle")
}

func TestValidateRejectsDuplicatePrerequisites(t *testing.T) {
	value := validFixture()
	value.Skills[1].Prerequisites = []string{value.Skills[0].ID, value.Skills[0].ID}
	assertInvalid(t, value, "repeats prerequisite")
}

func TestValidateRejectsMissingMilestoneSkill(t *testing.T) {
	value := validFixture()
	value.Milestones[0].SkillIDs[0] = "missing-skill"
	assertInvalid(t, value, "milestone")
}

func TestValidateRequiresEachSkillInOneMilestone(t *testing.T) {
	value := validFixture()
	value.Milestones[0].SkillIDs = value.Milestones[0].SkillIDs[:1]
	assertInvalid(t, value, "not assigned")
}

func TestValidateRejectsDuplicateSkillAcrossMilestones(t *testing.T) {
	value := validFixture()
	value.Milestones = append(value.Milestones, Milestone{
		ID:       "second",
		Title:    "Second milestone",
		SkillIDs: []string{value.Skills[0].ID},
	})
	assertInvalid(t, value, "appears in milestones")
}

func TestValidateRejectsDuplicateActivityIDs(t *testing.T) {
	value := validFixture()
	activity := ActivityReference{
		ID:       "activity-1",
		Kind:     ActivityMission,
		SkillIDs: []string{value.Skills[0].ID},
	}
	value.Activities = []ActivityReference{activity, activity}
	assertInvalid(t, value, "duplicate activity")
}

func TestValidateRejectsUnsupportedActivityKind(t *testing.T) {
	value := validFixture()
	value.Activities = []ActivityReference{{
		ID:       "activity-1",
		Kind:     "unknown",
		SkillIDs: []string{value.Skills[0].ID},
	}}
	assertInvalid(t, value, "unsupported kind")
}

func TestValidateRejectsMissingActivitySkill(t *testing.T) {
	value := validFixture()
	value.Activities = []ActivityReference{{
		ID:       "activity-1",
		Kind:     ActivityMission,
		SkillIDs: []string{"missing-skill"},
	}}
	assertInvalid(t, value, "activity")
}

func assertInvalid(t *testing.T, value Curriculum, message string) {
	t.Helper()
	err := value.Validate()
	if !errors.Is(err, ErrInvalidCurriculum) {
		t.Fatalf("Validate() error = %v, want ErrInvalidCurriculum", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Errorf("Validate() error = %q, want it to contain %q", err, message)
	}
}

func validFixture() Curriculum {
	return Curriculum{
		SchemaVersion: 1,
		Version:       "1.0.0",
		Title:         "Test curriculum",
		Skills: []Skill{
			{
				ID:         "skill-a",
				Title:      "Skill A",
				Objectives: []string{"Describe skill A"},
			},
			{
				ID:            "skill-b",
				Title:         "Skill B",
				Objectives:    []string{"Apply skill B"},
				Prerequisites: []string{"skill-a"},
			},
		},
		Milestones: []Milestone{{
			ID:       "milestone-a",
			Title:    "Milestone A",
			SkillIDs: []string{"skill-a", "skill-b"},
		}},
	}
}
