package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Orgs []Org
}

type Org struct {
	Name      string
	Workloads []Workload
}

type Workload struct {
	Name string
	Envs []Environment
}

type Environment struct {
	Name      string
	ProjectID string
}

func Parse(data []byte) (Config, error) {
	lines := strings.Split(string(data), "\n")
	var cfg Config

	var currentOrg *Org
	var currentWorkload *Workload
	var currentEnv *Environment
	section := ""

	for idx, raw := range lines {
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}

		indent := leadingSpaces(line)
		trimmed := strings.TrimSpace(line)

		switch {
		case indent == 0 && trimmed == "org:":
			section = "org-root"
		case indent == 2 && strings.HasPrefix(trimmed, "- name:"):
			if section != "org-root" && section != "org-item" {
				return Config{}, lineError(idx, "unexpected org entry")
			}
			name := valueAfter(trimmed, "- name:")
			if name == "" {
				return Config{}, lineError(idx, "org name cannot be empty")
			}
			cfg.Orgs = append(cfg.Orgs, Org{Name: name})
			currentOrg = &cfg.Orgs[len(cfg.Orgs)-1]
			currentWorkload = nil
			currentEnv = nil
			section = "org-item"
		case indent == 4 && trimmed == "workload:":
			if currentOrg == nil {
				return Config{}, lineError(idx, "workload section without org")
			}
			section = "workload-root"
		case indent == 6 && strings.HasPrefix(trimmed, "- name:"):
			if currentOrg == nil {
				return Config{}, lineError(idx, "workload entry without org")
			}
			if section != "workload-root" && section != "workload-item" {
				return Config{}, lineError(idx, "unexpected workload entry")
			}
			name := valueAfter(trimmed, "- name:")
			if name == "" {
				return Config{}, lineError(idx, "workload name cannot be empty")
			}
			currentOrg.Workloads = append(currentOrg.Workloads, Workload{Name: name})
			currentWorkload = &currentOrg.Workloads[len(currentOrg.Workloads)-1]
			currentEnv = nil
			section = "workload-item"
		case indent == 8 && trimmed == "env:":
			if currentWorkload == nil {
				return Config{}, lineError(idx, "env section without workload")
			}
			section = "env-root"
		case indent == 10 && strings.HasPrefix(trimmed, "- name:"):
			if currentWorkload == nil {
				return Config{}, lineError(idx, "env entry without workload")
			}
			if section != "env-root" && section != "env-item" {
				return Config{}, lineError(idx, "unexpected env entry")
			}
			name := valueAfter(trimmed, "- name:")
			if name == "" {
				return Config{}, lineError(idx, "env name cannot be empty")
			}
			currentWorkload.Envs = append(currentWorkload.Envs, Environment{Name: name})
			currentEnv = &currentWorkload.Envs[len(currentWorkload.Envs)-1]
			section = "env-item"
		case indent == 12 && strings.HasPrefix(trimmed, "project_id:"):
			if currentEnv == nil {
				return Config{}, lineError(idx, "project_id without env")
			}
			projectID := valueAfter(trimmed, "project_id:")
			if projectID == "" {
				return Config{}, lineError(idx, "project_id cannot be empty")
			}
			currentEnv.ProjectID = projectID
		default:
			return Config{}, lineError(idx, fmt.Sprintf("unsupported line %q", strings.TrimSpace(raw)))
		}
	}

	if len(cfg.Orgs) == 0 {
		return Config{}, fmt.Errorf("config does not define any org entries")
	}

	return cfg, nil
}

func (c Config) Resolve(orgName, workloadName, envName string) (string, error) {
	var orgMatches []Org
	for _, org := range c.Orgs {
		if org.Name == orgName {
			orgMatches = append(orgMatches, org)
		}
	}
	if len(orgMatches) == 0 {
		return "", fmt.Errorf("org %q not found in config", orgName)
	}
	if len(orgMatches) > 1 {
		return "", fmt.Errorf("org %q is duplicated in config", orgName)
	}

	var workloadMatches []Workload
	for _, workload := range orgMatches[0].Workloads {
		if workload.Name == workloadName {
			workloadMatches = append(workloadMatches, workload)
		}
	}
	if len(workloadMatches) == 0 {
		return "", fmt.Errorf("workload %q not found under org %q", workloadName, orgName)
	}
	if len(workloadMatches) > 1 {
		return "", fmt.Errorf("workload %q is duplicated under org %q", workloadName, orgName)
	}

	var envMatches []Environment
	for _, env := range workloadMatches[0].Envs {
		if env.Name == envName {
			envMatches = append(envMatches, env)
		}
	}
	if len(envMatches) == 0 {
		return "", fmt.Errorf("env %q not found under org %q workload %q", envName, orgName, workloadName)
	}
	if len(envMatches) > 1 {
		return "", fmt.Errorf("env %q is duplicated under org %q workload %q", envName, orgName, workloadName)
	}
	if strings.TrimSpace(envMatches[0].ProjectID) == "" {
		return "", fmt.Errorf("project_id is empty for org %q workload %q env %q", orgName, workloadName, envName)
	}

	return envMatches[0].ProjectID, nil
}

func stripComment(line string) string {
	var b strings.Builder
	inQuote := false
	for _, r := range line {
		switch {
		case r == '#':
			if !inQuote {
				return b.String()
			}
		case r == '"':
			inQuote = !inQuote
		}
		b.WriteRune(r)
	}
	return b.String()
}

func leadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func valueAfter(line, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, prefix))
}

func lineError(index int, message string) error {
	return fmt.Errorf("config parse error on line %d: %s", index+1, message)
}
