package curriculum

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const supportedSchemaVersion = 1

const (
	ActivityMission    = "mission"
	ActivityProject    = "project"
	ActivityAssessment = "assessment"
)

//go:embed v1.json
var defaultDocument []byte

var ErrInvalidCurriculum = errors.New("curriculum: invalid curriculum")

type Curriculum struct {
	SchemaVersion int                 `json:"schema_version"`
	Version       string              `json:"version"`
	Title         string              `json:"title"`
	Skills        []Skill             `json:"skills"`
	Milestones    []Milestone         `json:"milestones"`
	Activities    []ActivityReference `json:"activities"`
}

type Skill struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Objectives    []string `json:"objectives"`
	Prerequisites []string `json:"prerequisites"`
}

type Milestone struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	SkillIDs []string `json:"skill_ids"`
}

type ActivityReference struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	SkillIDs []string `json:"skill_ids"`
}

// Default returns the embedded, validated starter curriculum.
func Default() (Curriculum, error) {
	return Load(defaultDocument)
}

// Load decodes one curriculum document and validates its schema and references.
func Load(document []byte) (Curriculum, error) {
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()

	var value Curriculum
	if err := decoder.Decode(&value); err != nil {
		return Curriculum{}, fmt.Errorf("%w: decode document: %v", ErrInvalidCurriculum, err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Curriculum{}, fmt.Errorf("%w: document contains multiple JSON values", ErrInvalidCurriculum)
		}
		return Curriculum{}, fmt.Errorf("%w: decode trailing content: %v", ErrInvalidCurriculum, err)
	}

	if err := value.Validate(); err != nil {
		return Curriculum{}, err
	}
	return value, nil
}

// Validate checks curriculum identity, graph references, and prerequisite cycles.
func (value Curriculum) Validate() error {
	if value.SchemaVersion != supportedSchemaVersion {
		return invalid("unsupported schema_version %d", value.SchemaVersion)
	}
	if !validText(value.Version) || !validText(value.Title) {
		return invalid("version and title are required")
	}
	if len(value.Skills) == 0 {
		return invalid("at least one skill is required")
	}

	skills := make(map[string]struct{}, len(value.Skills))
	for _, skill := range value.Skills {
		if !validIdentifier(skill.ID) || !validText(skill.Title) {
			return invalid("skill id and title are required")
		}
		if _, exists := skills[skill.ID]; exists {
			return invalid("duplicate skill id %q", skill.ID)
		}
		skills[skill.ID] = struct{}{}

		if len(skill.Objectives) == 0 {
			return invalid("skill %q must have at least one objective", skill.ID)
		}
		for _, objective := range skill.Objectives {
			if !validText(objective) {
				return invalid("skill %q has an empty objective", skill.ID)
			}
		}
	}

	if len(value.Milestones) == 0 {
		return invalid("at least one milestone is required")
	}
	milestones := make(map[string]struct{}, len(value.Milestones))
	assignedSkills := make(map[string]string, len(value.Skills))
	for _, milestone := range value.Milestones {
		if !validIdentifier(milestone.ID) || !validText(milestone.Title) {
			return invalid("milestone id and title are required")
		}
		if _, exists := milestones[milestone.ID]; exists {
			return invalid("duplicate milestone id %q", milestone.ID)
		}
		milestones[milestone.ID] = struct{}{}
		if len(milestone.SkillIDs) == 0 {
			return invalid("milestone %q must reference at least one skill", milestone.ID)
		}

		for _, skillID := range milestone.SkillIDs {
			if _, exists := skills[skillID]; !exists {
				return invalid("milestone %q references missing skill %q", milestone.ID, skillID)
			}
			if previous, exists := assignedSkills[skillID]; exists {
				return invalid("skill %q appears in milestones %q and %q", skillID, previous, milestone.ID)
			}
			assignedSkills[skillID] = milestone.ID
		}
	}
	if len(assignedSkills) != len(skills) {
		for _, skill := range value.Skills {
			skillID := skill.ID
			if _, exists := assignedSkills[skillID]; !exists {
				return invalid("skill %q is not assigned to a milestone", skillID)
			}
		}
	}

	for _, skill := range value.Skills {
		seenPrerequisites := make(map[string]struct{}, len(skill.Prerequisites))
		for _, prerequisiteID := range skill.Prerequisites {
			if prerequisiteID == skill.ID {
				return invalid("skill %q cannot depend on itself", skill.ID)
			}
			if _, exists := skills[prerequisiteID]; !exists {
				return invalid("skill %q references missing prerequisite %q", skill.ID, prerequisiteID)
			}
			if _, exists := seenPrerequisites[prerequisiteID]; exists {
				return invalid("skill %q repeats prerequisite %q", skill.ID, prerequisiteID)
			}
			seenPrerequisites[prerequisiteID] = struct{}{}
		}
	}
	if err := validateAcyclic(value.Skills); err != nil {
		return err
	}

	activities := make(map[string]struct{}, len(value.Activities))
	for _, activity := range value.Activities {
		if !validIdentifier(activity.ID) {
			return invalid("activity id is required")
		}
		if _, exists := activities[activity.ID]; exists {
			return invalid("duplicate activity id %q", activity.ID)
		}
		activities[activity.ID] = struct{}{}
		if activity.Kind != ActivityMission && activity.Kind != ActivityProject && activity.Kind != ActivityAssessment {
			return invalid("activity %q has unsupported kind %q", activity.ID, activity.Kind)
		}
		if len(activity.SkillIDs) == 0 {
			return invalid("activity %q must reference at least one skill", activity.ID)
		}
		seenSkills := make(map[string]struct{}, len(activity.SkillIDs))
		for _, skillID := range activity.SkillIDs {
			if _, exists := skills[skillID]; !exists {
				return invalid("activity %q references missing skill %q", activity.ID, skillID)
			}
			if _, exists := seenSkills[skillID]; exists {
				return invalid("activity %q repeats skill %q", activity.ID, skillID)
			}
			seenSkills[skillID] = struct{}{}
		}
	}

	return nil
}

func validateAcyclic(skills []Skill) error {
	byID := make(map[string]Skill, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = skill
	}

	const (
		unvisited = iota
		visiting
		visited
	)
	states := make(map[string]int, len(skills))
	var visit func(string) error
	visit = func(skillID string) error {
		switch states[skillID] {
		case visiting:
			return invalid("prerequisite graph contains a cycle at skill %q", skillID)
		case visited:
			return nil
		}
		states[skillID] = visiting
		for _, prerequisiteID := range byID[skillID].Prerequisites {
			if err := visit(prerequisiteID); err != nil {
				return err
			}
		}
		states[skillID] = visited
		return nil
	}

	for _, skill := range skills {
		if err := visit(skill.ID); err != nil {
			return err
		}
	}
	return nil
}

func validIdentifier(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}

func validText(value string) bool {
	return strings.TrimSpace(value) != ""
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidCurriculum, fmt.Sprintf(format, args...))
}
